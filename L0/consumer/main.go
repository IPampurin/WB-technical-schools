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
	"sync/atomic"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
)

// выносим константы конфигурации по умолчанию, чтобы были на виду
const (
	topicNameConst       = "my-topic"   // имя топика, коррелируется с продюсером
	groupIDNameConst     = "my-groupID" // произвольное в нашем случае имя группы
	kafkaPortConst       = 9092         // порт, на котором сидит kafka по умолчанию
	limitConsumWorkConst = 10800        // время работы консумера по умолчанию в секундах (3 часа)
	servicePortConst     = 8081         // порт принимающего api-сервиса по умолчанию
	batchSizeConst       = 50           // количество сообщений в батче по умолчанию
	batchTimeoutMsConst  = 50           // время наполнения батча по умолчанию, мс
	maxRetriesConst      = 3            // количество повторных попыток связи по умолчанию
	retryDelayBaseConst  = 100          // базовая задержка для попыток связи по умолчанию
	clientTimeoutConst   = 30           // таймаут для HTTP клиента по умолчанию
)

var config atomic.Value // атомарное хранилище для конфигурации

// ConsumerConfig описывает настройки с учётом переменных окружения
type ConsumerConfig struct {
	Topic           string        // имя топика (коррелируется с продюсером)
	GroupID         string        // имя группы
	KafkaPort       int           // порт, на котором сидит kafka
	LimitConsumWork time.Duration // время работы консумера в секундах
	ServicePort     int           // порт принимающего api-сервиса
	BatchSize       int           // количество сообщений в батче
	BatchTimeout    time.Duration // время наполнения батча, мс
	MaxRetries      int           // количество повторных попыток связи
	RetryDelayBase  time.Duration // базовая задержка для попыток связи
	ClientTimeout   time.Duration // таймаут для HTTP клиента
}

// getEnvString проверяет наличие и корректность переменной окружения (строковое значение)
func getEnvString(envVariable, defaultValue string) string {

	value, ok := os.LookupEnv(envVariable)
	if ok {
		return value
	}

	return defaultValue
}

// getEnvInt проверяет наличие и корректность переменной окружения (числовое значение)
func getEnvInt(envVariable string, defaultValue int) int {

	value, ok := os.LookupEnv(envVariable)
	if ok {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
		log.Printf("ошибка парсинга %s, используем значение по умолчанию: %d", envVariable, defaultValue)
	}

	return defaultValue
}

// readConfig уточняет конфигурацию с учётом переменных окружения,
// проверяет переменные окружения и устанавливает параметры работы
func readConfig() *ConsumerConfig {

	return &ConsumerConfig{
		Topic:           getEnvString("TOPIC_NAME_STR", topicNameConst),
		GroupID:         getEnvString("GROUP_ID_NAME_STR", groupIDNameConst),
		KafkaPort:       getEnvInt("KAFKA_PORT_NUM", kafkaPortConst),
		LimitConsumWork: time.Duration(getEnvInt("TIME_LIMIT_CONSUMER_S", limitConsumWorkConst)) * time.Second,
		ServicePort:     getEnvInt("SERVICE_PORT_NUM", servicePortConst),
		BatchSize:       getEnvInt("BATCH_SIZE_NUM", batchSizeConst),
		BatchTimeout:    time.Duration(getEnvInt("BATCH_TIMEOUT_MS", batchTimeoutMsConst)) * time.Millisecond,
		MaxRetries:      getEnvInt("MAX_RETRIES_NUM", maxRetriesConst),
		RetryDelayBase:  time.Duration(getEnvInt("RETRY_DELEY_BASE_MS", retryDelayBaseConst)) * time.Millisecond,
		ClientTimeout:   time.Duration(getEnvInt("CLIENT_TIMEOUT_S", clientTimeoutConst)) * time.Second,
	}
}

// getConfig безопасно получает конфигурацию
func getConfig() *ConsumerConfig {

	if cfg := config.Load(); cfg != nil {
		return cfg.(*ConsumerConfig)
	}
	// возвращаем конфигурацию по умолчанию, если ещё не инициализировано
	return &ConsumerConfig{
		Topic:           topicNameConst,
		GroupID:         groupIDNameConst,
		KafkaPort:       kafkaPortConst,
		LimitConsumWork: time.Duration(limitConsumWorkConst) * time.Second,
		ServicePort:     servicePortConst,
		BatchSize:       batchSizeConst,
		BatchTimeout:    time.Duration(batchTimeoutMsConst) * time.Millisecond,
		MaxRetries:      maxRetriesConst,
		RetryDelayBase:  time.Duration(retryDelayBaseConst) * time.Millisecond,
		ClientTimeout:   time.Duration(clientTimeoutConst) * time.Second,
	}
}

// updateConfig обновляет конфигурацию (для hot reload в будущем)
func updateConfig(newConfig *ConsumerConfig) {

	config.Store(newConfig)
}

// consumer это основной код консумера
func consumer(ctx context.Context, cfg *ConsumerConfig) error {

	// ридер из кафки
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{fmt.Sprintf("localhost:%d", cfg.KafkaPort)},
		Topic:    cfg.Topic,
		GroupID:  cfg.GroupID,
		MinBytes: 10000,  // минимальный пакет 10 КБ (3-5 сообщений)
		MaxBytes: 500000, // максимальный пакет батчей 500 КБ (150-200 сообщений)
		MaxWait:  cfg.BatchTimeout,
	})
	defer func() {
		if err := r.Close(); err != nil {
			log.Printf("ошибка при закрытии ридера Kafka: %v", err)
		}
	}()

	log.Printf("Консумер подписан на топик '%s' в группе '%s'\n", r.Config().Topic, r.Config().GroupID)
	log.Println("Начинаем вычитывать !!!")

	// канал для передачи батчей
	batches := make(chan []kafka.Message, 10)

	// запускаем воркеры обработки батчей
	workerCount := 5
	for i := 0; i < workerCount; i++ {
		go batchWorker(ctx, r, batches, cfg, i)
	}

	// читаем и собираем сообщения в батчи
	err := collectBatches(ctx, r, batches, cfg)
	if err != nil {
		return fmt.Errorf("ошибка сбора батчей: %w", err)
	}

	return nil
}

// sendBatchWithRetry передает сообщение в API с повторами
func sendBatchWithRetry(ctx context.Context, apiUrl string, data []byte) (*http.Response, error) {

	// организуем клиента для отправки вычитанных из кафки сообщений на api сервиса
	httpClient := &http.Client{
		Timeout: clientTimeoutConst,
	}

	for attempt := 1; attempt <= maxRetriesConst; attempt++ {

		// шлём запрос на сервер и получаем ответ
		resp, err := httpClient.Post(apiUrl, "application/json", bytes.NewReader(data))
		if err == nil {
			return resp, nil // успешная отправка (закрываем resp.Body далее при обработке)
		}

		log.Printf("Ошибка сети (попытка %d/%d): %v", attempt, maxRetriesConst, err)

		// проверяем номер попытки и делаем паузу перед следующей попыткой
		if attempt < maxRetriesConst {

			// высчитываем увеличивающуюся паузу (200ms, 600ms, 1200ms)
			delay := retryDelayBaseConst * time.Duration(attempt*attempt+attempt)
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

	return nil, fmt.Errorf("не удалось отправить запрос после %d попыток", maxRetriesConst)
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

func batchWorker(ctx context.Context, r *kafka.Reader, batches <-chan []kafka.Message, setConfig *ConsumerConfig, workerID int) {

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

func collectBatches(ctx context.Context, r *kafka.Reader, batches chan<- []kafka.Message, cfg *ConsumerConfig) error {

	currentBatch := make([]kafka.Message, 0, cfg.BatchSize)
	timer := time.NewTimer(cfg.BatchTimeout)

loop:
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
				currentBatch = make([]kafka.Message, 0, cfg.BatchSize)
			}
			timer.Reset(cfg.BatchTimeout)

		default:
			// Чтение с коротким таймаутом чтобы не блокировать select
			msg, err := r.ReadMessage(100 * time.Millisecond)
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					continue // Нет сообщений - продолжаем
				}
				if errors.Is(err, context.Canceled) {
					break loop // Graceful shutdown
				}
				log.Printf("Ошибка чтения из Kafka: %v", err)
				continue
			}

			// Добавляем в батч
			currentBatch = append(currentBatch, msg)
			if len(currentBatch) >= cfg.BatchSize {
				batches <- currentBatch
				currentBatch = make([]kafka.Message, 0, cfg.BatchSize)
				timer.Reset(cfg.BatchTimeout)
			}
		}
	}

	return nil
}

func main() {

	// считываем конфигурацию
	cfg := readConfig()

	// сохраняем в atomic.Value (для будущего hot reload)
	updateConfig(cfg)

	// apiUrl := fmt.Sprintf("http://localhost:%d/order", servicePort)

	// заведём контекст для отмены работы консумера
	ctx, cancel := context.WithTimeout(context.Background(), cfg.LimitConsumWork)
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
	err := consumer(ctx, cfg)
	if err != nil {
		log.Printf("консумер завершился с критической ошибкой: %v", err)
		os.Exit(1)
	}

	log.Println("Консумер корректно завершил работу.")
}
