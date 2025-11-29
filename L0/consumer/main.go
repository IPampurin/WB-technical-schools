package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
)

var (
	kafkaPortConst       = 9092                   // порт, на котором сидит kafka по умолчанию
	limitConsumWorkConst = 10800                  // время работы консумера по умолчанию в секундах (3 часа)
	servicePortConst     = 8081                   // порт принимающего api-сервиса по умолчанию
	batchSizeConst       = 50                     // количество сообщений в батче по умолчанию
	batchTimeoutMsConst  = 1000                   // время наполнения батча по умолчанию, мс
	topicConst           = "my-topic-L0"          // имя топика коррелируется с продюсером
	groupIDConst         = "my-groupID"           // произвольное в нашем случае имя группы
	maxRetries           = 3                      // количество повторных попыток связи
	retryDelayBase       = 100 * time.Millisecond // базовая задержка для попыток связи
	clientTimeout        = 30 * time.Second       // таймаут для HTTP клиента
)

// readSettings проверяет переменные окружения и устанавливает параметры работы
func readSettings() (int, int, string, int, time.Duration) {

	// по умолчанию слушаем брокер на порту kafkaPortConst
	kafkaPortStr, ok := os.LookupEnv("KAFKA_PORT")
	if !ok {
		kafkaPortStr = fmt.Sprintf("%d", kafkaPortConst)
	}
	kafkaPort, err := strconv.Atoi(kafkaPortStr)
	if err != nil {
		log.Printf("ошибка парсинга kafkaPort, используем значение по умолчанию: %d", kafkaPortConst)
		kafkaPort = kafkaPortConst
	}

	// по умолчанию время работы консумера принимаем
	limitConsumWorkStr, ok := os.LookupEnv("TIME_LIMIT_CONSUMER")
	if !ok {
		limitConsumWorkStr = fmt.Sprintf("%d", limitConsumWorkConst)
	}
	timeLimitConsumer, err := strconv.Atoi(limitConsumWorkStr)
	if err != nil {
		log.Printf("ошибка парсинга timeLimitConsumer, используем значение по умолчанию: %d", limitConsumWorkConst)
		timeLimitConsumer = limitConsumWorkConst
	}

	// по умолчанию порт хоста для сервиса servicePortConst (доступ в браузере на localhost:servicePortConst)
	servicePortStr, ok := os.LookupEnv("SERVICE_PORT")
	if !ok {
		servicePortStr = fmt.Sprintf("%d", servicePortConst)
	}
	servicePort, err := strconv.Atoi(servicePortStr)
	if err != nil {
		log.Printf("ошибка парсинга servicePort, используем значение по умолчанию: %d", servicePortConst)
		servicePort = servicePortConst
	}
	// определяем url для HTTP клиента
	apiUrl := fmt.Sprintf("http://localhost:%d/order", servicePort)

	// по умолчанию количество сообщений в батче == batchSizeConst
	batchSizeStr, ok := os.LookupEnv("BATCH_SIZE")
	if !ok {
		batchSizeStr = fmt.Sprintf("%d", batchSizeConst)
	}
	batchSize, err := strconv.Atoi(batchSizeStr)
	if err != nil {
		log.Printf("ошибка парсинга batchSize, используем значение по умолчанию: %d", batchSizeConst)
		batchSize = batchSizeConst
	}

	// по умолчанию время наполнения батча сообщениями == batchTimeoutMsConst миллисекунд
	batchTimeoutStr, ok := os.LookupEnv("BATCH_TIMEOUT_MS")
	if !ok {
		batchTimeoutStr = fmt.Sprintf("%d", batchTimeoutMsConst)
	}
	batchTimeoutMs, err := strconv.Atoi(batchTimeoutStr)
	if err != nil {
		log.Printf("ошибка парсинга batchTimeoutMs, используем значение по умолчанию: %dms", batchTimeoutMsConst)
		batchTimeoutMs = batchTimeoutMsConst
	}
	batchTimeout := time.Duration(batchTimeoutMs) * time.Millisecond

	log.Printf("настройки батчирования: размер=%d, таймаут=%v", batchSize, batchTimeout)

	return kafkaPort, timeLimitConsumer, apiUrl, batchSize, batchTimeout
}

// consumer это основной код консумера
func consumer(ctx context.Context, kafkaPort int, apiUrl string, batchSize int, batchTimeout time.Duration) error {

	// ридер из кафки
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{fmt.Sprintf("localhost:%d", kafkaPort)},
		Topic:    topicConst,
		GroupID:  groupIDConst,
		MinBytes: 0,
		MaxWait:  10 * time.Second,
	})
	defer func() {
		if err := r.Close(); err != nil {
			log.Printf("ошибка при закрытии ридера Kafka: %v", err)
		}
	}()

	log.Printf("Консумер подписан на топик '%s' в группе '%s'\n", topicConst, groupIDConst)
	log.Println("Начинаем вычитывать !!!")

	// канал для передачи батчей
	batches := make(chan []kafka.Message, 10)

	// запускаем воркеры обработки батчей
	workerCount := 5
	for i := 0; i < workerCount; i++ {
		go batchWorker(ctx, r, batches, apiUrl, i)
	}

	// собираем сообщения в батчи
	err := collectBatches(ctx, r, batches, batchSize, batchTimeout)
	if err != nil {
		return fmt.Errorf("ошибка сбора батчей: %w", err)
	}

	return nil
}

// sendBatchWithRetry передает сообщение в API с повторами
func sendBatchWithRetry(ctx context.Context, apiUrl string, data []byte) (*http.Response, error) {

	// организуем клиента для отправки вычитанных из кафки сообщений на api сервиса
	httpClient := &http.Client{
		Timeout: clientTimeout,
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {

		// шлём запрос на сервер и получаем ответ
		resp, err := httpClient.Post(apiUrl, "application/json", bytes.NewReader(data))
		if err == nil {
			return resp, nil // успешная отправка (закрываем resp.Body далее при обработке)
		}

		log.Printf("Ошибка сети (попытка %d/%d): %v", attempt, maxRetries, err)

		// проверяем номер попытки и делаем паузу перед следующей попыткой
		if attempt < maxRetries {

			// высчитываем увеличивающуюся паузу (200ms, 600ms, 1200ms)
			delay := retryDelayBase * time.Duration(attempt*attempt+attempt)
			log.Printf("Попытка отправки %v", attempt)

			select {
			case <-time.After(delay):
				// продолжаем следующую попытку
			case <-ctx.Done():
				log.Println("Прерываем повторы отправки сообщений при отмене контекста.")
				return nil, ctx.Err()
			}
		}
	}

	return nil, fmt.Errorf("не удалось отправить запрос после %d попыток", maxRetries)
}

// handleAPIResponse обрабатывает ответ API и решает коммитить ли сообщение
func handleAPIResponse(ctx context.Context, r *kafka.Reader, m kafka.Message, resp *http.Response) (bool, error) {

	defer resp.Body.Close()

	switch {

	// успешные ответы - коммитим
	case resp.StatusCode >= 200 && resp.StatusCode < 300: // 200-299
		fallthrough
	case resp.StatusCode == http.StatusConflict: // 409 - дубль
		if err := r.CommitMessages(ctx, m); err != nil {
			if errors.Is(err, context.Canceled) {
				return false, fmt.Errorf("Прерываем коммит сообщения при отмене контекста: %w", err)
			}
			return false, fmt.Errorf("ошибка коммита: %w", err)
		}
		log.Printf("Сообщение обработано и закоммичено: %s (статус: %d)", m.Key, resp.StatusCode)
		return true, nil

		// редиректы - не коммитим, пробуем снова
	case resp.StatusCode >= 300 && resp.StatusCode < 400: // 300-399
		log.Printf("Получен редирект: %d для %s", resp.StatusCode, m.Key)
		return false, nil

	// клиентские ошибки - не коммитим, пропускаем сообщение
	case resp.StatusCode >= 400 && resp.StatusCode < 500: // 400-499 (кроме 409)
		log.Printf("клиентская ошибка: %d для %s", resp.StatusCode, m.Key)
		return false, nil

	// серверные ошибки - критические
	case resp.StatusCode >= 500 && resp.StatusCode < 600: // 500-599
		return false, fmt.Errorf("серверная ошибка API: статус %d", resp.StatusCode)

	// неведомый статус
	default:
		return false, fmt.Errorf("неизвестный статус API: %d", resp.StatusCode)
	}
}

func batchWorker(ctx context.Context, r *kafka.Reader, batches <-chan []kafka.Message, apiUrl string, workerID int) {

	for batch := range batches {
		// Подготовка данных для API
		var payload [][]byte
		for _, msg := range batch {
			payload = append(payload, msg.Value)
		}

		// Отправка с retry
		results, err := sendBatchWithRetry(ctx, apiUrl, payload)
		// ... обработка результатов
	}
}

func collectBatches(ctx context.Context, r *kafka.Reader, batches chan<- []kafka.Message, batchSize int, batchTimeout time.Duration) error {

	var currentBatch []kafka.Message
	timer := time.NewTimer(batchTimeout)

	for {
		select {
		case <-ctx.Done():
			// Отправляем последний батч при завершении
			if len(currentBatch) > 0 {
				batches <- currentBatch
			}
			close(batches) // Сигнал воркерам о завершении
			return nil

		case <-timer.C:
			// Таймаут - отправляем накопленное
			if len(currentBatch) > 0 {
				batches <- currentBatch
				currentBatch = make([]kafka.Message, 0, batchSize)
			}
			timer.Reset(batchTimeout)

		default:
			// Чтение с коротким таймаутом чтобы не блокировать select
			msg, err := r.ReadMessage(100 * time.Millisecond)
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					continue // Нет сообщений - продолжаем
				}
				if errors.Is(err, context.Canceled) {
					return nil // Graceful shutdown
				}
				log.Printf("Ошибка чтения из Kafka: %v", err)
				continue
			}

			// Добавляем в батч
			currentBatch = append(currentBatch, msg)
			if len(currentBatch) >= batchSize {
				batches <- currentBatch
				currentBatch = make([]kafka.Message, 0, batchSize)
				timer.Reset(batchTimeout)
			}
		}
	}

	return nil
}

func main() {

	kafkaPort, timeLimitConsumer, apiUrl, batchSize, batchTimeout := readSettings()

	// заведём контекст для отмены работы консумера
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeLimitConsumer)*time.Second)
	defer cancel()

	// обработка сигналов ОС для graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// ждём сигнал отмены в фоне
	go func() {
		<-sigChan
		log.Println("Получен сигнал остановки, завершаем работу...")
		cancel()
	}()

	// запускаем основной код консумера
	err := consumer(ctx, kafkaPort, apiUrl, batchSize, batchTimeout)
	if err != nil {
		log.Printf("консумер завершился с критической ошибкой: %v", err)
		os.Exit(1)
	}

	log.Println("Консумер корректно завершил работу.")
}
