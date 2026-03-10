package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/IPampurin/ImageProcessor/pkg/configuration"
	"github.com/IPampurin/ImageProcessor/pkg/domain"
	"github.com/IPampurin/ImageProcessor/pkg/processor/worker"
	"github.com/IPampurin/ImageProcessor/pkg/s3"
	"github.com/wb-go/wbf/kafka"
	"github.com/wb-go/wbf/logger"
	"github.com/wb-go/wbf/retry"

	kafkago "github.com/segmentio/kafka-go"
)

func main() {

	// cоздаём контекст
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// запускаем горутину обработки сигналов
	go signalHandler(ctx, cancel)

	// считываем .env файл
	cfg, err := configuration.ReadConfig()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// настраиваем логгер
	appLogger, err := logger.InitLogger(
		logger.ZapEngine,
		"ImageProcessor",
		os.Getenv("APP_ENV"),
		logger.WithLevel(logger.InfoLevel),
	)
	if err != nil {
		log.Fatalf("Ошибка создания логгера: %v", err)
	}
	defer func() { _ = appLogger.(*logger.ZapAdapter) }()

	// подключаемся к S3 (MinIO)
	s3Client, err := s3.InitS3(ctx, &cfg.S3, appLogger)
	if err != nil {
		appLogger.Error("ошибка подключения к S3 хранилищу", "error", err)
		return
	}

	// инициализируем Kafka
	brokerAddr := fmt.Sprintf("%s:%d", cfg.Kafka.HostName, cfg.Kafka.Port)
	brokers := []string{brokerAddr}
	consumer := kafka.NewConsumer(brokers, cfg.Kafka.InputTopic, cfg.Kafka.ConsumerGroup)
	defer consumer.Close()

	producer := kafka.NewProducer(brokers, cfg.Kafka.OutputTopic)
	defer producer.Close()

	// каналы для пайплайна
	tasksCh := make(chan kafkago.Message)
	resultsCh := make(chan *processorResult)

	// запускаем consumer в фоне с ретраями
	consumer.StartConsuming(ctx, tasksCh, retry.Strategy{
		Attempts: 3,
		Delay:    100 * time.Millisecond,
	})

	// создаём воркер
	imageWorker := worker.New(s3Client, appLogger, cfg.Thumb.ThumbWidth, cfg.Thumb.ThumbHeight)

	// запускаем пул воркеров
	const numWorkers = 5
	for i := 0; i < numWorkers; i++ {
		go startWorker(ctx, imageWorker, tasksCh, resultsCh, appLogger)
	}

	// запускаем отправитель результатов (передаём consumer для коммита)
	go startResultSender(ctx, consumer, producer, resultsCh, appLogger)

	appLogger.Info("Процессор обработки изображений успешно запущен")

	// ожидаем сигнала завершения
	<-ctx.Done()

	appLogger.Info("Завершение работы процессора")

	time.Sleep(2 * time.Second)
}

// processorResult связывает исходное сообщение Kafka и результат его обработки
type processorResult struct {
	rawMsg kafkago.Message
	result *domain.ImageResult
	err    error
}

// startWorker читает задачи из tasksCh, обрабатывает и отправляет результаты в resultsCh
func startWorker(ctx context.Context, w *worker.Worker, tasksCh <-chan kafkago.Message, resultsCh chan<- *processorResult, log logger.Logger) {

	for {
		select {
		case <-ctx.Done():
			return
		case rawMsg := <-tasksCh:
			var task domain.ImageTask
			if err := json.Unmarshal(rawMsg.Value, &task); err != nil {
				log.Error("ошибка разбора задачи JSON", "error", err, "offset", rawMsg.Offset)
				resultsCh <- &processorResult{rawMsg: rawMsg, err: err}
				continue
			}

			log.Info("обработка задачи", "imageID", task.ImageID, "offset", rawMsg.Offset)
			result, err := w.ProcessTask(ctx, &task)
			resultsCh <- &processorResult{rawMsg: rawMsg, result: result, err: err}
		}
	}
}

// startResultSender получает результаты, отправляет их в Kafka и коммитит исходные сообщения
func startResultSender(ctx context.Context, consumer *kafka.Consumer, producer *kafka.Producer, resultsCh <-chan *processorResult, log logger.Logger) {

	for {
		select {
		case <-ctx.Done():
			return
		case pres := <-resultsCh:

			// если произошла ошибка обработки, формируем результат со статусом "failed"
			if pres.err != nil {
				errMsg := pres.err.Error()
				failedResult := &domain.ImageResult{
					ImageID:      "unknown", // imageID может быть неизвестен, если ошибка при разборе JSON
					Status:       "failed",
					ErrorMessage: &errMsg,
					Variants:     nil,
				}

				// пытаемся извлечь imageID из задачи, если она была частично разобрана
				var task domain.ImageTask
				if json.Unmarshal(pres.rawMsg.Value, &task) == nil {
					failedResult.ImageID = task.ImageID
				}

				// отправляем failed-результат
				data, _ := json.Marshal(failedResult)
				if err := producer.Send(ctx, []byte(failedResult.ImageID), data); err != nil {
					log.Error("ошибка отправки failed-результата в Kafka", "error", err)
				}

			} else if pres.result != nil {

				// успешный результат — отправляем как есть
				data, err := json.Marshal(pres.result)
				if err != nil {
					log.Error("ошибка маршалинга результата", "error", err, "imageID", pres.result.ImageID)
				} else {
					if err := producer.Send(ctx, []byte(pres.result.ImageID), data); err != nil {
						log.Error("ошибка отправки результата в Kafka", "error", err, "imageID", pres.result.ImageID)
					}
				}
			}

			// коммитим исходное сообщение (после отправки результата)
			if err := consumer.Commit(ctx, pres.rawMsg); err != nil {
				log.Error("ошибка коммита сообщения Kafka", "error", err, "offset", pres.rawMsg.Offset)
			}
		}
	}
}

// signalHandler обрабатывает сигналы отмены
func signalHandler(ctx context.Context, cancel context.CancelFunc) {

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	select {
	case <-ctx.Done():
		return
	case <-sigChan:
		cancel()
		return
	}
}
