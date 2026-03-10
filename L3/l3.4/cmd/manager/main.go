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
	"github.com/IPampurin/ImageProcessor/pkg/manager/db"
	"github.com/IPampurin/ImageProcessor/pkg/manager/server"
	"github.com/IPampurin/ImageProcessor/pkg/manager/service"
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
		os.Getenv("APP_ENV"), // пока оставим пустым
		logger.WithLevel(logger.InfoLevel),
	)
	if err != nil {
		log.Fatalf("Ошибка создания логгера: %v", err)
	}
	defer func() { _ = appLogger.(*logger.ZapAdapter) }()

	// получаем экземпляр БД
	storageDB, err := db.InitDB(ctx, &cfg.DB, appLogger)
	if err != nil {
		appLogger.Error("ошибка подключения к БД", "error", err)
		return
	}
	defer func() { _ = db.CloseDB(storageDB) }()

	// получаем экземпляр S3-совместимого хранилища
	storageS3, err := s3.InitS3(ctx, &cfg.S3, appLogger)
	if err != nil {
		appLogger.Error("ошибка подключения к внешнему S3 хранилищу", "error", err)
		return
	}
	defer func() { _ = s3.CloseS3(storageS3) }()

	// инициализируем Kafka
	brokerAddr := fmt.Sprintf("%s:%d", cfg.Kafka.HostName, cfg.Kafka.Port)
	brokers := []string{brokerAddr}
	producer := kafka.NewProducer(brokers, cfg.Kafka.InputTopic)
	defer producer.Close()

	consumer := kafka.NewConsumer(brokers, cfg.Kafka.OutputTopic, cfg.Kafka.ConsumerGroup)
	defer consumer.Close()

	// получаем экземпляр слоя бизнес-логики
	service := service.InitService(ctx, storageDB, storageS3, cfg.Kafka.InputTopic)

	// запускаем релизер outbox
	go startOutboxRelay(ctx, storageDB, producer, appLogger)

	// запускаем consumer результатов
	go startResultConsumer(ctx, consumer, service, appLogger)

	// запускаем сервер
	err = server.Run(ctx, &cfg.Server, service, appLogger)
	if err != nil {
		appLogger.Error("Ошибка сервера", "error", err)
		cancel()
		return
	}

	appLogger.Info("Приложение корректно завершено")
}

// startOutboxRelay периодически читает outbox и отправляет сообщения в Kafka
func startOutboxRelay(ctx context.Context, db *db.DataBase, producer *kafka.Producer, log logger.Logger) {

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("остановка outbox relay")
			return
		case <-ticker.C:
			outboxList, err := db.GetUnsentOutbox(ctx, 100)
			if err != nil {
				log.Error("ошибка получения outbox записей", "error", err)
				continue
			}
			for _, out := range outboxList {
				err := producer.Send(ctx, []byte(out.Key), out.Payload)
				if err != nil {
					log.Error("ошибка отправки сообщения в Kafka", "error", err, "outboxID", out.ID)
					continue
				}
				if err := db.DeleteOutbox(ctx, out.ID); err != nil {
					log.Error("ошибка удаления outbox записи", "error", err, "outboxID", out.ID)
				} else {
					log.Info("отправлено и удалено из outbox", "outboxID", out.ID, "key", out.Key)
				}
			}
		}
	}
}

// startResultConsumer читает результаты обработки из Kafka и обновляет БД
func startResultConsumer(ctx context.Context, consumer *kafka.Consumer, svc *service.Service, log logger.Logger) {

	msgCh := make(chan kafkago.Message) // используем тип из kafkago
	consumer.StartConsuming(ctx, msgCh, retry.Strategy{
		Attempts: 3,
		Delay:    100 * time.Millisecond,
	})

	for msg := range msgCh {
		var result domain.ImageResult
		if err := json.Unmarshal(msg.Value, &result); err != nil {
			log.Error("ошибка десериализации результата", "error", err)
			_ = consumer.Commit(ctx, msg)
			continue
		}

		if err := svc.ProcessResult(ctx, &result, log); err != nil {
			log.Error("ошибка обработки результата", "error", err, "imageID", result.ImageID)
			_ = consumer.Commit(ctx, msg)
			continue
		}

		if err := consumer.Commit(ctx, msg); err != nil {
			log.Error("ошибка коммита", "error", err)
		}
	}
}

// signalHandler обрабатывет сигналы отмены
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
