package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
)

// выносим константы конфигурации по умолчанию, чтобы были на виду

const (
	topicNameConst       = "my-topic"     // имя топика, коррелируется с продюсером
	groupIDNameConst     = "my-groupID"   // произвольное в нашем случае имя группы
	kafkaPortConst       = 9092           // порт, на котором сидит kafka по умолчанию
	limitConsumWorkConst = 10800          // время работы консумера по умолчанию в секундах (3 часа)
	servicePortConst     = 8081           // порт принимающего api-сервиса по умолчанию
	batchSizeConst       = 50             // количество сообщений в батче по умолчанию
	batchTimeoutMsConst  = 50             // время наполнения батча по умолчанию, мс
	maxRetriesConst      = 3              // количество повторных попыток связи по умолчанию
	retryDelayBaseConst  = 100            // базовая задержка для попыток связи по умолчанию
	clientTimeoutConst   = 30             // таймаут для HTTP клиента по умолчанию
	dlqTopicConst        = "my-topic-DLQ" // топик для DLQ
	workersCountConst    = 5              // количество параллельных обработчиков в пайплайне
)

// OrderResponse структура для ответов из api (копия из postOrders.go)
type OrderResponse struct {
	OrderUID     string `json:"orderUID"`
	Status       string `json:"status"` // "success", "conflict", "badRequest", "error"
	Message      string `json:"message,omitempty"`
	ShouldCommit bool   `json:"shouldCommit"`
	ShouldDLQ    bool   `json:"shouldDLQ"`
}

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
	DlqTopic        string        // топик для DLQ
	WorkersCount    int           // количество параллельных обработчиков в пайплайне
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
		DlqTopic:        getEnvString("DLQ_TOPIC_NAME_STR", dlqTopicConst),
		WorkersCount:    getEnvInt("WORKERS_COUNT", workersCountConst),
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
		DlqTopic:        dlqTopicConst,
		WorkersCount:    workersCountConst,
	}
}

// updateConfig обновляет конфигурацию (для hot reload в будущем)
func updateConfig(newConfig *ConsumerConfig) {

	config.Store(newConfig)
}

// consumer это основной код консумера
func consumer(ctx context.Context, cfg *ConsumerConfig) error {

	// устанавливаем соединение с брокером для автосоздания DLQ
	conn, err := kafka.DialLeader(context.Background(), "tcp", fmt.Sprintf("localhost:%d", cfg.KafkaPort), cfg.DlqTopic, 0)
	if err != nil {
		log.Fatalf("ошибка создания DLQ-топика кафки: %v\n", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Ошибка при закрытии консумером соединения с кафкой: %v", err)
		}
	}()

	// врайтер для DLQ
	dlqWriter := &kafka.Writer{
		Addr:  kafka.TCP(fmt.Sprintf("kafka:%d", cfg.KafkaPort)),
		Topic: cfg.DlqTopic,
	}
	defer func() {
		if err := dlqWriter.Close(); err != nil {
			log.Printf("ошибка при закрытии dlqWriter в консумере: %v", err)
		}
	}()

	// ридер из кафки
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{fmt.Sprintf("localhost:%d", cfg.KafkaPort)},
		Topic:    cfg.Topic,
		GroupID:  cfg.GroupID,
		MinBytes: 10000,  // минимальный пакет 10 КБ (3-5 сообщений)
		MaxBytes: 500000, // максимальный пакет батчей 500 КБ (ориентировочно 150-200 сообщений)
		MaxWait:  cfg.BatchTimeout,
	})
	defer func() {
		if err := r.Close(); err != nil {
			log.Printf("ошибка при закрытии ридера Kafka: %v", err)
		}
	}()

	log.Printf("Консумер подписан на топик '%s' в группе '%s'.\n", r.Config().Topic, r.Config().GroupID)
	log.Printf("DLQ writer консумера подписан на топик '%s'.\n", dlqWriter.Topic)
	log.Println("Начинаем вычитывать !!!")

	// TODO поиграть с размером буферов исходя из ожидаемой пропускной способности пайплайна
	messages := make(chan kafka.Message, cfg.BatchSize*cfg.WorkersCount) // канал для входящих сообщений с большими буферами
	batches := make(chan []kafka.Message, cfg.WorkersCount*2)            // канал для передачи батчей
	// errCh := make(chan error, 1)                                         // канал для передачи ошибки ридера при чтении сообщений из кафки

	// фоном обрабатываем ответы api, коммитим или заполняем DLQ
	for i := 0; i < workersCountConst; i++ {
		// TODO сделать	go processBatchResponse()
	}

	// запускаем фоновые воркеры обработки батчей (отправка с ретраем и получение ответа api)
	for i := 0; i < workersCountConst; i++ {
		// TODO сделать	go batchWorker(ctx, r, dlqWriter, batches, cfg, i)
	}

	// фоном комплектуем батчи из прочитанных сообщений
	go complectBatches(messages, batches, cfg)

	// читаем сообщения из кафки
	err = readMsgOfKafka(ctx, r, messages, cfg)
	if err != nil {
		return fmt.Errorf("ошибка сбора батчей: %w", err)
	}

	return nil
}

// batchWorker направляет в api батчи сообщений и получает ответы от api
func batchWorker(r *kafka.Reader, dlqWriter *kafka.Writer, batches <-chan []kafka.Message, cfg *ConsumerConfig) {

	apiURL := fmt.Sprintf("http://localhost:%d/order", cfg.ServicePort)

	// организуем клиента для отправки вычитанных из кафки сообщений на api сервиса
	httpClient := &http.Client{
		Timeout: cfg.ClientTimeout,
	}

	for batch := range batches {

		// нулевых батчей приходить не должно, но на всякий случай проверяем
		if len(batch) == 0 {
			continue
		}

		log.Printf("Воркер %d: обработка батча из %d сообщений", workerID, len(batch))

		// подготавливаем данные для API
		var jsonMessages []json.RawMessage
		messageMap := make(map[string]kafka.Message) // order_uid -> message

		for _, msg := range batch {
			jsonMessages = append(jsonMessages, msg.Value)
			// извлекаем order_uid для маппинга
			if orderUID := extractOrderUID(msg.Value); orderUID != "" {
				messageMap[orderUID] = msg
			}
		}

		// Отправляем батч с повторами
		response, err := sendBatchWithRetry(httpClient, apiURL, jsonMessages, cfg)
		if err != nil {
			log.Printf("Воркер %d: ошибка отправки батча: %v", workerID, err)
			// При критической ошибке отправляем все в DLQ
			for _, msg := range batch {
				sendToDLQ(dlqWriter, msg, err.Error())
			}
			continue
		}

		// Обрабатываем ответы
		processBatchResponse(r, dlqWriter, batch, response, messageMap, workerID)
	}
}

// sendBatchWithRetry передает сообщение в API с повторами
func sendBatchWithRetry(ctx context.Context, client *http.Client, apiURL string, data []json.RawMessage, cfg *ConsumerConfig) ([]OrderResponse, error) {

	requestBody, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации: %w", err)
	}

	for attempt := 1; attempt <= cfg.MaxRetries; attempt++ {

		req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(requestBody))
		if err != nil {
			return nil, fmt.Errorf("ошибка создания запроса: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Ошибка сети (попытка %d/%d): %v", attempt, maxRetriesConst, err)

			// проверяем номер попытки и делаем паузу перед следующей попыткой
			if attempt < cfg.MaxRetries {

				// высчитываем увеличивающуюся паузу (200ms, 600ms, 1200ms)
				delay := cfg.RetryDelayBase * time.Duration(attempt*attempt+attempt) * time.Millisecond
				log.Printf("Попытка отправки %v", attempt)

				select {
				case <-time.After(delay):
					// продолжаем следующую попытку
				case <-ctx.Done():
					log.Println("Прерываем повторы отправки сообщений при отмене контекста.")
					return nil, ctx.Err()
				}
			}
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode == http.StatusMultiStatus || resp.StatusCode == http.StatusCreated {
			var responses []OrderResponse
			if err := json.NewDecoder(resp.Body).Decode(&responses); err != nil {
				return nil, fmt.Errorf("ошибка декодирования ответа: %w", err)
			}
			return responses, nil
		}

		// если статус не 201 или 207, пробуем снова
		log.Printf("Попытка %d/%d: неожиданный статус %d", attempt, cfg.MaxRetries, resp.StatusCode)
		resp.Body.Close()
	}

	return nil, fmt.Errorf("не удалось отправить запрос после %d попыток", maxRetriesConst)
}

func processBatchResponse(r *kafka.Reader, dlqWriter *kafka.Writer, batch []kafka.Message, responses []OrderResponse, messageMap map[string]kafka.Message, workerID int) {

	// Группируем сообщения по решению
	var toCommit []kafka.Message
	var toDLQ []kafka.Message
	var toSkip []kafka.Message

	for _, resp := range responses {
		msg, exists := messageMap[resp.OrderUID]
		if !exists {
			continue
		}

		switch resp.Status {
		case "success", "conflict":
			if resp.ShouldCommit {
				toCommit = append(toCommit, msg)
				log.Printf("Воркер %d: заказ %s - %s", workerID, resp.OrderUID, resp.Status)
			}
		case "bad_request", "error":
			if resp.ShouldDLQ {
				toDLQ = append(toDLQ, msg)
				log.Printf("Воркер %d: заказ %s в DLQ: %s", workerID, resp.OrderUID, resp.Message)
			}
		default:
			toSkip = append(toSkip, msg)
			log.Printf("Воркер %d: заказ %s пропущен: %s", workerID, resp.OrderUID, resp.Status)
		}
	}

	// Коммитим успешные
	if len(toCommit) > 0 {
		if err := r.CommitMessages(ctx, toCommit...); err != nil {
			log.Printf("Воркер %d: ошибка коммита: %v", workerID, err)
		} else {
			log.Printf("Воркер %d: успешно закоммичено %d сообщений", workerID, len(toCommit))
		}
	}

	// Отправляем в DLQ
	for _, msg := range toDLQ {
		sendToDLQ(ctx, dlqWriter, msg, "ошибка обработки")
	}

	log.Printf("Воркер %d: обработан батч. Успешно: %d, DLQ: %d, Пропущено: %d",
		workerID, len(toCommit), len(toDLQ), len(toSkip))
}

func sendToDLQ(ctx context.Context, writer *kafka.Writer, msg kafka.Message, reason string) {

	dlqMsg := kafka.Message{
		Key:   []byte(fmt.Sprintf("dlq-%s", string(msg.Key))),
		Value: msg.Value,
		Headers: []kafka.Header{
			{Key: "original-topic", Value: []byte(msg.Topic)},
			{Key: "original-partition", Value: []byte(fmt.Sprintf("%d", msg.Partition))},
			{Key: "original-offset", Value: []byte(fmt.Sprintf("%d", msg.Offset))},
			{Key: "error-reason", Value: []byte(reason)},
			{Key: "timestamp", Value: []byte(time.Now().Format(time.RFC3339))},
		},
	}

	if err := writer.WriteMessages(ctx, dlqMsg); err != nil {
		log.Printf("ошибка отправки в DLQ: %v", err)
	}
}

func extractOrderUID(data []byte) string {

	var temp struct {
		OrderUID string `json:"order_uid"`
	}
	if err := json.Unmarshal(data, &temp); err == nil {
		return temp.OrderUID
	}
	return ""
}

// complectBatches фоном читает канал messages, собирает батчи и наполняет канал batches,
// контекст не используем в complectBatches в надежде обработать все сообщения из канала messages
func complectBatches(messages <-chan kafka.Message, batches chan<- []kafka.Message, cfg *ConsumerConfig) {

	var wg sync.WaitGroup

	// запускаем воркеры сбора батчей
	for i := 0; i < workersCountConst; i++ {
		wg.Add(1)
		go func() {

			defer wg.Done()

			currentBatch := make([]kafka.Message, 0, cfg.BatchSize)
			ticker := time.NewTicker(cfg.BatchTimeout)
			defer ticker.Stop()

			for {
				select {
				// если прошло время, выделенное на сбор батча, отправляем что накопилось и повторяем вычитывание
				case <-ticker.C:
					if len(currentBatch) > 0 {
						batches <- currentBatch
						currentBatch = currentBatch[:0:cfg.BatchSize]
					}
					continue
				// если можем считать сообщение и канал открыт, читаем сообщение и копим батч
				case msg, ok := <-messages:
					if !ok {
						if len(currentBatch) > 0 {
							batches <- currentBatch
						}
						return
					}
					currentBatch = append(currentBatch, msg)
					// если достигли нужного размера сразу отправляем
					if len(currentBatch) >= cfg.BatchSize {
						batches <- currentBatch
						currentBatch = currentBatch[:0:cfg.BatchSize]
					}
				}
			}
		}()
	}

	// ждём завершения всех батчесобирателей в отдельной горутине
	go func() {
		wg.Wait()
		close(batches)
	}()
}

// readMsgOfKafka читает сообщения из кафки и наполняет канал messages
func readMsgOfKafka(ctx context.Context, r *kafka.Reader, messages chan kafka.Message, cfg *ConsumerConfig) error {

	timeForReadMessage := 50 * time.Millisecond   // время блокировки на чтении
	retryForReadMessage := 100 * time.Millisecond // пауза при не критических ошибках

	defer close(messages)

	for {
		ctxReadMessage, cancelRM := context.WithTimeout(ctx, timeForReadMessage)

		msg, err := r.ReadMessage(ctxReadMessage)
		cancelRM()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			if errors.Is(err, context.DeadlineExceeded) {
				select {
				case <-time.After(retryForReadMessage): // добавляем паузу при таймауте
					continue
				case <-ctx.Done():
					return nil
				}
			}
			return err
		}

		select {
		case messages <- msg:
		case <-ctx.Done():
			return nil
		}
	}
}

func main() {

	// считываем конфигурацию
	cfg := readConfig()

	// сохраняем в atomic.Value (для будущего hot reload)
	updateConfig(cfg)

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
		cancel()
		os.Exit(1)
	}

	log.Println("Консумер корректно завершил работу.")
}
