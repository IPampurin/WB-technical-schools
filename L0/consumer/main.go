package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	maxRetriesConst      = 3              // количество повторных попыток отправки батчей в api по умолчанию
	retryDelayBaseConst  = 100            // базовая задержка для попыток отправки по умолчанию
	clientTimeoutConst   = 30             // таймаут для HTTP клиента по умолчанию
	dlqTopicConst        = "my-topic-DLQ" // топик для DLQ
	workersCountConst    = 5              // количество параллельных обработчиков в пайплайне
)

// OrderResponse структура для ответов из api (копия из postOrders.go)
type OrderResponse struct {
	OrderUID     string `json:"orderUID"`          // идентификатор сообщения
	Status       string `json:"status"`            // статус: "success", "conflict", "badRequest", "error"
	Message      string `json:"message,omitempty"` // информация об ошибке
	ShouldCommit bool   `json:"shouldCommit"`      // можно ли коммитить в кафке
	ShouldDLQ    bool   `json:"shouldDLQ"`         // надо ли отправить в DLQ
}

// BatchInfo объединяет информацию об ответах api по сообщениям с самими сообщениями
type BatchInfo struct {
	respOfBatch  []OrderResponse          // ответы по каждому из сообщений батча
	messageByUID map[string]kafka.Message // мапа идентификации [orderUID]kafka.Message
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

/*
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
*/

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

	// клиент для отправки вычитанных из кафки сообщений на api сервиса
	httpClient := &http.Client{
		Timeout: cfg.ClientTimeout,
	}

	log.Printf("Консумер подписан на топик '%s' в группе '%s'.\n", r.Config().Topic, r.Config().GroupID)
	log.Printf("DLQ writer консумера подписан на топик '%s'.\n", dlqWriter.Topic)
	log.Println("Начинаем вычитывать !!!")

	// TODO поиграть с размером буферов исходя из ожидаемой пропускной способности пайплайна
	messages := make(chan kafka.Message, cfg.BatchSize*cfg.WorkersCount) // канал для входящих сообщений с большим буфером
	batches := make(chan []kafka.Message, cfg.WorkersCount*2)            // канал для передачи батчей
	responses := make(chan BatchInfo, cfg.WorkersCount*2)                // канал передачи ответов по батчам и мап с сообщениями
	// errCh := make(chan error, 1)                                         // канал для передачи ошибки ридера при чтении сообщений из кафки

	var wgPipe sync.WaitGroup

	// 4. обрабатываем ответы api для каждого сообщения, коммитим или заполняем DLQ
	wgPipe.Add(1)
	go processBatchResponse(r, dlqWriter, responses, cfg, &wgPipe)

	// 3. отправляем батчи с ретраем и получаем ответы api
	wgPipe.Add(1)
	go batchWorker(dlqWriter, httpClient, batches, responses, cfg, &wgPipe)

	// 2. фоном комплектуем батчи из прочитанных сообщений
	wgPipe.Add(1)
	go complectBatches(messages, batches, cfg, &wgPipe)

	// ждём окончания обработки всех считанных readMsgOfKafka сообщений
	go func() {
		wgPipe.Wait()
	}()

	// 1. читаем сообщения из кафки
	err = readMsgOfKafka(ctx, r, messages, cfg)
	if err != nil {
		return fmt.Errorf("ошибка сбора батчей: %w", err)
	}

	return nil
}

// processBatchResponse обрабатывает полученные от api ответы по каждому сообщению из батча
func processBatchResponse(r *kafka.Reader, dlqWriter *kafka.Writer, responses <-chan BatchInfo, cfg *ConsumerConfig, wgPipe *sync.WaitGroup) {

	defer wgPipe.Done()

	var wg sync.WaitGroup

	// запускаем пул воркеров
	for i := 0; i < cfg.WorkersCount; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			toCommit := 0 // счётчик закоммиченых воркером сообщений
			toDLQ := 0    // счётчик сообщений, отправленных в DLQ

			// слушаем канал с информацией об ответах api, пока канал открыт
			for batchInfo := range responses {

				// для обработки ответа по каждому сообщению смотрим есть ли
				// сообщение в мапе и можно ли его закоммитить
				for _, resp := range batchInfo.respOfBatch {

					msg, ok := batchInfo.messageByUID[resp.OrderUID]
					if !ok {
						// не смотря на такое невероятное стечение обстоятельств, чтобы не останавливать конвейер, логируем и продолжаем
						log.Printf("processBatchResponse: воркер %d: в мапе заказов не оказалось сообщения с OrderUID = %s !!!\n", i, resp.OrderUID)
						continue
					}

					switch {
					case resp.ShouldCommit:
						if err := r.CommitMessages(context.Background(), msg); err != nil {
							log.Printf("processBatchResponse: воркер %d: ошибка коммита сообщения %s: %v", i, string(msg.Key), err)
						} else {
							toCommit++
						}
					case resp.ShouldDLQ:
						sendToDLQ(dlqWriter, msg, resp.Message)
						toDLQ++
					default:
						log.Printf("processBatchResponse: воркер %d: в структуре ответа по сообщению с OrderUID = %s не хватает информации!!!\n", i, resp.OrderUID)
					}
				}

				log.Printf("processBatchResponse: воркер %d: обработан батч. Успешно: %d, DLQ: %d.", i, toCommit, toDLQ)
			}
		}(i)
	}

	// ждём завершения воркеров
	go func() {
		wg.Wait()
		log.Println("processBatchResponse: все воркеры завершены.")
	}()
}

// batchWorker получает батчи и направляет в api, ответы передаёт далее на обработку
func batchWorker(dlqWriter *kafka.Writer, httpClient *http.Client, batches <-chan []kafka.Message,
	responses chan<- BatchInfo, cfg *ConsumerConfig, wgPipe *sync.WaitGroup) {

	defer wgPipe.Done()

	var wg sync.WaitGroup

	// запускаем пул воркеров
	for i := 0; i < cfg.WorkersCount; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			batchCounter := 0 // счётчик пришедших батчей
			lenBatches := 0   // счётчик обработанных сообщений
			counterDLQ := 0   // количество сообщений, отправленных в DLQ
			counterResp := 0  // количество ответов по сообщениям (если всё ок, то counterResp = counterDLQ + lenBatches)

			// слушаем канал с группами сообщений
			for batch := range batches {

				// нулевых батчей приходить не должно, но на всякий случай проверяем
				if len(batch) == 0 {
					continue
				}

				batchCounter++
				lenBatches += len(batch)

				// 1. подготавливаем данные для API

				var jsonMessages []json.RawMessage
				messageMap := make(map[string]kafka.Message) // [orderUID]->message

				for _, msg := range batch {
					// извлекаем orderUID для маппинга
					if orderUID := extractOrderUID(msg.Value); orderUID != "" {
						messageMap[orderUID] = msg
						jsonMessages = append(jsonMessages, msg.Value)
					} else {
						// при отсутствии идентификатора сообщение шлём в DLQ
						counterDLQ++
						sendToDLQ(dlqWriter, msg, "not exist OrderUID")
					}
				}

				// 2. отправляем батч с повторами

				// полученный ответ это []OrderResponse, в котором orderUID-ы это ключи для мапы [orderUID]->message
				response, err := sendBatchWithRetry(httpClient, jsonMessages, cfg)
				if err != nil {
					log.Printf("batchWorker: воркер %d: ошибка отправки батча: %v", i, err)
					// при критической ошибке отправляем все в DLQ и идём за новой порцией сообщений
					for _, msg := range batch {
						counterDLQ++
						sendToDLQ(dlqWriter, msg, err.Error())
					}
					continue
				}

				counterResp += len(response)

				// 3. объединяем ответ по батчу и мапу [orderUID]->message в структуру
				//    и шлём в канал для обработки в processBatchResponse

				batchInfo := BatchInfo{
					respOfBatch:  response,
					messageByUID: messageMap,
				}
				responses <- batchInfo
			}

			log.Printf("batchWorker: воркер %d: обработано %d батчей, из %d сообщений, сообщений в DLQ %d, ответов api %d.\n", i, batchCounter, lenBatches, counterDLQ, counterResp)
		}(i)
	}

	// по завершению воркеров закрываем канал, чтобы по мере обработки сообщений
	// из канала завершили работу последующие этапы обработки
	go func() {
		wg.Wait()
		close(responses)
		log.Println("batchWorker: все воркеры завершены, канал responses закрыт.")
	}()
}

// extractOrderUID вытаскивает OrderUID из msg.Value,
// для последующей идентификации msg
func extractOrderUID(data []byte) string {

	var msgIdentificator struct {
		OrderUID string `json:"order_uid"`
	}
	if err := json.Unmarshal(data, &msgIdentificator); err == nil {
		return msgIdentificator.OrderUID
	}
	// если по какому-то чудесному стечению обстоятельств OrderUID отсутствует
	// в информации о заказе, то такое сообщение годится только для DLQ
	return ""
}

// sendToDLQ уточняет заголовок сообщения и отправляет его в DLQ
func sendToDLQ(w *kafka.Writer, msg kafka.Message, reason string) {

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

	// используем пустой контекст с целью обработать все сообщения, которые вычитал ридер в readMsgOfKafka
	if err := w.WriteMessages(context.Background(), dlqMsg); err != nil {
		log.Printf("ошибка отправки сообщения %s в DLQ: %v", string(msg.Key), err)
	}
}

// sendBatchWithRetry передает сообщение в API с повторами
func sendBatchWithRetry(client *http.Client, data []json.RawMessage, cfg *ConsumerConfig) ([]OrderResponse, error) {

	// определяем адрес отправки
	apiURL := fmt.Sprintf("http://localhost:%d/order", cfg.ServicePort)

	// формируем тело запроса
	requestBody, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации: %w", err)
	}

	for attempt := 1; attempt <= cfg.MaxRetries; attempt++ {

		// направляем запрос
		resp, err := client.Post(apiURL, "application/json", bytes.NewReader(requestBody))
		if err != nil {
			log.Printf("Ошибка сети (попытка %d/%d): %v", attempt, cfg.MaxRetries, err)

			// проверяем номер попытки и делаем паузу перед следующей попыткой
			if attempt < cfg.MaxRetries {

				// делаем увеличивающуюся паузу (200ms, 600ms, 1200ms)
				delay := cfg.RetryDelayBase * time.Duration(attempt*attempt+attempt) * time.Millisecond
				time.Sleep(delay)
			}
			continue
		}

		// читаем тело ответа
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("Ошибка чтения ответа (попытка %d/%d): %v", attempt, cfg.MaxRetries, err)
			continue
		}

		// проверяем код ответа и парсим ответ
		if resp.StatusCode == http.StatusMultiStatus || resp.StatusCode == http.StatusCreated {
			var responses []OrderResponse
			if err := json.Unmarshal(body, &responses); err != nil {
				return nil, fmt.Errorf("ошибка декодирования ответа: %w", err)
			}
			return responses, nil
		}

		// если статус не 201 или 207, пробуем снова
		log.Printf("Попытка %d/%d: неожиданный статус %d", attempt, cfg.MaxRetries, resp.StatusCode)
	}

	return nil, fmt.Errorf("не удалось отправить запрос после %d попыток", cfg.MaxRetries)
}

// complectBatches читает канал messages, собирает батчи и наполняет канал batches,
// контекст не используем в complectBatches в надежде обработать все сообщения из канала messages
func complectBatches(messages <-chan kafka.Message, batches chan<- []kafka.Message, cfg *ConsumerConfig, wgPipe *sync.WaitGroup) {

	defer wgPipe.Done()

	var wg sync.WaitGroup

	// запускаем пул воркеров
	for i := 0; i < cfg.WorkersCount; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			currentBatch := make([]kafka.Message, 0, cfg.BatchSize) // батч с сообщениями
			ticker := time.NewTicker(cfg.BatchTimeout)              // таймер для отключения комплектования батча по времени
			defer ticker.Stop()

			lenCounter := 0   // счётчик количества сообщений в отправленных батчах
			batchCounter := 0 // счётчик количества отправленных батчей

			sendBatch := func() {
				copyBatch := make([]kafka.Message, len(currentBatch), len(currentBatch))
				copy(copyBatch, currentBatch)
				batches <- copyBatch
				currentBatch = currentBatch[:0:cfg.BatchSize]
				lenCounter += len(copyBatch)
				batchCounter++
			}

			for {
				select {
				// если прошло время, выделенное на сбор батча, отправляем что накопилось, и повторяем вычитывание
				case <-ticker.C:
					if len(currentBatch) > 0 {
						sendBatch()
					}
					continue
				// если можем считать сообщение и канал открыт, читаем сообщение и копим батч
				case msg, ok := <-messages:
					if !ok {
						if len(currentBatch) > 0 {
							sendBatch()
						}
						log.Printf("complectBatches: канал входящих сообщений закрыт, воркер %d завершил работу, отправлено %d сообщений в %d батчах.\n", i, lenCounter, batchCounter)
						return // если канал закрыт и читать больше нечего, завершаем работу
					}
					currentBatch = append(currentBatch, msg)
					// если достигли нужного размера сразу отправляем
					if len(currentBatch) >= cfg.BatchSize {
						sendBatch()
					}
				}
			}
		}(i)
	}

	// по завершению воркеров закрываем канал, чтобы по мере обработки сообщений
	// из канала завершили работу последующие этапы обработки
	go func() {
		wg.Wait()
		close(batches)
		log.Println("complectBatches: все воркеры завершены, канал batches закрыт.")
	}()
}

// readMsgOfKafka читает сообщения из кафки и наполняет канал messages
func readMsgOfKafka(ctx context.Context, r *kafka.Reader, messages chan<- kafka.Message, cfg *ConsumerConfig) error {

	// закрываем при выходе канал, чтобы по мере обработки
	// сообщений из канала завершили работу последующие этапы обработки
	defer close(messages)

	timeForReadMessage := 50 * time.Millisecond   // время блокировки на чтении сообщения
	retryForReadMessage := 100 * time.Millisecond // пауза при не критических ошибках

	inCounter := 0  // счётчик входящих сообщений для логирования
	outCounter := 0 // счётчик отправленных в канал сообщений
	for {
		// объявляем контекст для ReadMessage
		ctxReadMessage, cancelRM := context.WithTimeout(ctx, timeForReadMessage)

		msg, err := r.ReadMessage(ctxReadMessage)
		cancelRM()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Printf("readMsgOfKafka: считано %d сообщений, отправлено %d сообщений.\n", inCounter, outCounter)
				return nil
			}
			if errors.Is(err, context.DeadlineExceeded) {
				select {
				// добавляем паузу при таймауте: вероятно, сообщений нет и грузить CPU нет смысла
				case <-time.After(retryForReadMessage):
					continue
				case <-ctx.Done():
					log.Printf("readMsgOfKafka: считано %d сообщений, отправлено %d сообщений.\n", inCounter, outCounter)
					return nil
				}
			}
			log.Printf("readMsgOfKafka: считано %d сообщений, отправлено %d сообщений.\n", inCounter, outCounter)
			return err
		}

		inCounter++

		select {
		case messages <- msg:
			outCounter++
		case <-ctx.Done():
			log.Printf("readMsgOfKafka: считано %d сообщений, отправлено %d сообщений.\n", inCounter, outCounter)
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
