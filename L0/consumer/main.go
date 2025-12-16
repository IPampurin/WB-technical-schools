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
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
)

// выносим константы конфигурации по умолчанию, чтобы были на виду.
// для работы программы менять в .env
const (
	topicNameConst      = "my-topic"     // имя топика, коррелируется с продюсером
	groupIDNameConst    = "my-groupID"   // произвольное в нашем случае имя группы
	kafkaHostConst      = "kafka"        // имя службы (контейнера) в сети докера по умолчанию
	kafkaPortConst      = 9092           // порт, на котором сидит kafka по умолчанию
	serviceHostConst    = "service"      // имя службы (контейнера) в сети докера по умолчанию
	servicePortConst    = 8081           // порт принимающего api-сервиса по умолчанию
	batchSizeConst      = 1000           // количество сообщений в батче по умолчанию
	batchTimeoutConst   = 5              // время наполнения батча по умолчанию, с
	maxRetriesConst     = 3              // количество повторных попыток отправки батчей в api по умолчанию
	retryDelayBaseConst = 100            // базовая задержка для попыток отправки по умолчанию
	clientTimeoutConst  = 30             // таймаут для HTTP клиента по умолчанию
	dlqTopicConst       = "my-topic-DLQ" // топик для DLQ
)

// OrderResponse структура для ответов из api (копия из postOrders.go)
type OrderResponse struct {
	OrderUID string `json:"order_uid"`         // идентификатор сообщения
	Status   string `json:"status"`            // статус: "success", "conflict", "badRequest", "error"
	Message  string `json:"message,omitempty"` // информация об ошибке
}

// BatchInfo объединяет информацию об ответах api по сообщениям с самими сообщениями
type BatchInfo struct {
	respOfBatch      []OrderResponse           // ответы по каждому из сообщений батча
	messageByUID     map[string]*kafka.Message // мапа идентификации [orderUID]kafka.Message
	lastBatchMessage *kafka.Message            // указатель на последнее сообщение в батче, чтобы коммитить весь батч, так как порядок сообщений сохраняется
}

// ConsumerConfig описывает настройки с учётом переменных окружения
type ConsumerConfig struct {
	Topic          string        // имя топика (коррелируется с продюсером)
	GroupID        string        // имя группы
	KafkaHost      string        // имя службы (контейнера) в сети докера
	KafkaPort      int           // порт, на котором сидит kafka
	ServiceHost    string        // имя службы (контейнера) в сети докера
	ServicePort    int           // порт принимающего api-сервиса
	BatchSize      int           // количество сообщений в батче
	BatchTimeout   time.Duration // время наполнения батча, с
	MaxRetries     int           // количество повторных попыток связи
	RetryDelayBase time.Duration // базовая задержка для попыток связи
	ClientTimeout  time.Duration // таймаут для HTTP клиента
	DlqTopic       string        // топик для DLQ
}

var cfg *ConsumerConfig

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
		Topic:          getEnvString("TOPIC_NAME_STR", topicNameConst),
		GroupID:        getEnvString("GROUP_ID_NAME_STR", groupIDNameConst),
		KafkaHost:      getEnvString("KAFKA_HOST_NAME", kafkaHostConst),
		KafkaPort:      getEnvInt("KAFKA_PORT_NUM", kafkaPortConst),
		ServiceHost:    getEnvString("SERVICE_HOST_NAME", serviceHostConst),
		ServicePort:    getEnvInt("SERVICE_PORT_NUM", servicePortConst),
		BatchSize:      getEnvInt("BATCH_SIZE_NUM", batchSizeConst),
		BatchTimeout:   time.Duration(getEnvInt("BATCH_TIMEOUT_S", batchTimeoutConst)) * time.Second,
		MaxRetries:     getEnvInt("MAX_RETRIES_NUM", maxRetriesConst),
		RetryDelayBase: time.Duration(getEnvInt("RETRY_DELEY_BASE_MS", retryDelayBaseConst)) * time.Millisecond,
		ClientTimeout:  time.Duration(getEnvInt("CLIENT_TIMEOUT_S", clientTimeoutConst)) * time.Second,
		DlqTopic:       getEnvString("DLQ_TOPIC_NAME_STR", dlqTopicConst),
	}
}

// consumer это основной код консумера
func consumer(ctx context.Context, errCh chan<- error, endCh chan struct{}) {

	// устанавливаем соединение с брокером для автосоздания DLQ
	conn, err := kafka.DialLeader(context.Background(), "tcp", fmt.Sprintf("%s:%d", cfg.KafkaHost, cfg.KafkaPort), cfg.DlqTopic, 0)
	if err != nil {
		log.Printf("ошибка создания DLQ-топика кафки: %v\n", err)
		errCh <- fmt.Errorf("ошибка создания DLQ-топика кафки: %v", err)
		// разблокируем main() и выходим
		close(errCh)
		close(endCh)
		return
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Ошибка при закрытии консумером соединения с кафкой: %v", err)
		}
	}()

	// врайтер для DLQ
	dlqWriter := &kafka.Writer{
		Addr:  kafka.TCP(fmt.Sprintf("%s:%d", cfg.KafkaHost, cfg.KafkaPort)),
		Topic: cfg.DlqTopic,
	}
	defer func() {
		if err := dlqWriter.Close(); err != nil {
			log.Printf("ошибка при закрытии dlqWriter в консумере: %v", err)
		}
	}()

	// ридер из кафки
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{fmt.Sprintf("%s:%d", cfg.KafkaHost, cfg.KafkaPort)},
		Topic:    cfg.Topic,
		GroupID:  cfg.GroupID,
		MinBytes: 10000,  // минимальный пакет
		MaxBytes: 500000, // максимальный пакет батчей
		// MaxWait:  cfg.BatchTimeout,
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

	// размер буферов каналов следует назначать исходя из сетевых задержек и ожидаемой пропускной
	// способности пайплайна. Интересно: есть ли какая-то формула или практический подход?
	// Слишком большой размер буферов приводит к длительному grace периоду при остановке контейнера.
	messages := make(chan *kafka.Message, cfg.BatchSize*10) // канал для входящих сообщений с большим буфером
	batches := make(chan []*kafka.Message, cfg.BatchSize/4) // канал для передачи батчей
	responses := make(chan *BatchInfo, cfg.BatchSize/4)     // канал передачи ответов по батчам и мап с сообщениями

	// wgPipe для ожидания всех горутин конвейера
	var wgPipe sync.WaitGroup

	// 1. читаем сообщения из кафки
	wgPipe.Add(1)
	go readMsgOfKafka(ctx, r, messages, errCh, &wgPipe)

	// 2. комплектуем батчи из прочитанных сообщений
	wgPipe.Add(1)
	go complectBatches(messages, batches, &wgPipe)

	// 3. отправляем батчи с ретраем и получаем ответы api с информацией по каждому сообщению
	wgPipe.Add(1)
	go batchWorker(dlqWriter, httpClient, batches, responses, &wgPipe)

	// 4. обрабатываем ответы api для каждого сообщения - коммитим или заполняем DLQ
	wgPipe.Add(1)
	go processBatchResponse(r, dlqWriter, responses, endCh, &wgPipe)

	// ждём окончания обработки всех считанных readMsgOfKafka сообщений
	wgPipe.Wait()

	// close(errCh) уже выполнился при выходе из readMsgOfKafka
	// close(endCh) уже выполнился при выходе из processBatchResponse
}

// readMsgOfKafka читает сообщения из кафки и наполняет канал messages
func readMsgOfKafka(ctx context.Context, r *kafka.Reader, messages chan<- *kafka.Message, errCh chan<- error, wgPipe *sync.WaitGroup) {

	start := time.Now()

	defer wgPipe.Done()

	defer func() {
		// закрываем при выходе канал messages, чтобы по мере обработки
		// сообщений из канала завершили работу последующие этапы обработки
		close(messages)
		log.Println("readMsgOfKafka: закрыл канал messages.")

		// закрываем при выходе канал errCh, чтобы разблокировать main()
		close(errCh)
		log.Println("readMsgOfKafka: закрыл канал errCh.")
	}()

	inCounter := 0  // счётчик входящих сообщений для логирования
	outCounter := 0 // счётчик отправленных в канал сообщений
	for {

		msg, err := r.FetchMessage(ctx)
		if err != nil {
			// если контекст отменили (graceful shutdown)
			if errors.Is(err, context.Canceled) {
				log.Printf("readMsgOfKafka: чтение из kafka завершено, получено %d сообщений, отправлено на батчирование %d сообщений, за %v с.\n",
					inCounter, outCounter, time.Since(start).Seconds())
				errCh <- nil // оповещаем main() и выходим, конвейер продолжает обработку уже вычитанных сообщений
				return
			}
			log.Printf("readMsgOfKafka: чтение из kafka завершено, получено %d сообщений, отправлено на батчирование %d сообщений, за %v с.\n",
				inCounter, outCounter, time.Since(start).Seconds())
			errCh <- err // оповещаем main() и выходим, конвейер продолжает обработку уже вычитанных сообщений
			return
		}

		inCounter++
		messages <- &msg
		outCounter++

		if inCounter%10000 == 0 {
			log.Printf("readMsgOfKafka: получено %d сообщений, отправлено на батчирование %d сообщений, за %v с.\n",
				inCounter, outCounter, time.Since(start).Seconds())
		}
	}
}

// complectBatches собирает батчи из сообщений из messages (по количеству или по времени) и наполняет канал batches,
// контекст не используем в надежде обработать все уже поступившие сообщения из канала messages
func complectBatches(messages <-chan *kafka.Message, batches chan<- []*kafka.Message, wgPipe *sync.WaitGroup) {

	defer wgPipe.Done()

	// закрываем при выходе канал batches, чтобы по мере обработки
	// сообщений из канала завершили работу последующие этапы обработки
	defer func() {
		close(batches)
		log.Println("complectBatches: закрыл канал batches.")
	}()

	start := time.Now()

	currentBatch := make([]*kafka.Message, 0, cfg.BatchSize) // батч с указателями на сообщения
	ticker := time.NewTicker(cfg.BatchTimeout)               // таймер для отключения комплектования батча по времени
	defer ticker.Stop()

	inCounter := 0    // счётчик входящих сообщений
	lenCounter := 0   // счётчик количества сообщений в отправленных батчах
	batchCounter := 0 // счётчик количества отправленных батчей

	sendBatch := func() {
		copyBatch := make([]*kafka.Message, len(currentBatch))
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
			if !ok { // если канал закрыт и читать больше нечего, завершаем работу
				if len(currentBatch) > 0 {
					sendBatch()
				}
				log.Printf("complectBatches: канал messages закрыт, получено %d сообщений, отправлено %d сообщений в %d батчах, время работы этапа %v c.\n",
					inCounter, lenCounter, batchCounter, time.Since(start).Seconds())
				return
			}
			inCounter++
			currentBatch = append(currentBatch, msg)
			// если достигли нужного размера сразу отправляем
			if len(currentBatch) >= cfg.BatchSize {
				sendBatch()
			}

			if inCounter%10000 == 0 {
				log.Printf("complectBatches: получено %d сообщений, отправлено %d сообщений в %d батчах, за %v c.\n",
					inCounter, lenCounter, batchCounter, time.Since(start).Seconds())
			}
		}
	}
}

// batchWorker получает батчи и направляет в api, ответы передаёт далее на обработку в канал responses
func batchWorker(dlqWriter *kafka.Writer, httpClient *http.Client, batches <-chan []*kafka.Message,
	responses chan<- *BatchInfo, wgPipe *sync.WaitGroup) {

	defer wgPipe.Done()

	// закрываем при выходе канал responses, чтобы по мере обработки
	// сообщений из канала завершили работу последующие этапы обработки
	defer func() {
		close(responses)
		log.Println("batchWorker: закрыл канал responses.")
	}()

	start := time.Now()

	batchCounter := 0   // счётчик пришедших батчей
	lenBatches := 0     // счётчик обработанных сообщений
	counterDLQ := 0     // количество сообщений, отправленных в DLQ
	respCounter := 0    // количество ответов по запросам
	counterRespMsg := 0 // количество ответов по сообщениям (если всё ок, то counterResp = counterDLQ + lenBatches)

	// слушаем канал с группами сообщений
	for batch := range batches {

		// нулевых батчей приходить не должно, но на всякий случай проверяем
		if len(batch) == 0 {
			log.Printf("batchWorker: следующим после батча %d пришёл батч с нулевой длинной.", batchCounter)
			continue
		}

		batchCounter++
		lenBatches += len(batch)

		// 1. подготавливаем данные для API

		var jsonMessages []json.RawMessage
		messageMap := make(map[string]*kafka.Message) // [orderUID]->*message

		for i := range batch {
			// извлекаем orderUID для маппинга
			if orderUID := extractOrderUID(batch[i].Value); orderUID != "" {
				messageMap[orderUID] = batch[i]
				jsonMessages = append(jsonMessages, batch[i].Value)
			} else {
				// при отсутствии идентификатора сообщение шлём в DLQ
				counterDLQ++
				sendToDLQ(dlqWriter, batch[i], "not exist OrderUID")
			}
		}

		// 2. отправляем батч с повторами

		// полученный ответ это []OrderResponse, в котором orderUID-ы это ключи для мапы [orderUID]->message
		response, err := sendBatchWithRetry(httpClient, jsonMessages)
		if err != nil {
			// если api отпало, то сообщения идут в DLQ - возможно, стоит предусмотреть "аварийный" топик на этот случай
			log.Printf("batchWorker: ошибка отправки батча: %v", err)
			// при критической ошибке отправляем все в DLQ и идём за новой порцией сообщений
			for i := range batch {
				counterDLQ++
				sendToDLQ(dlqWriter, batch[i], err.Error())
			}
			continue
		}

		respCounter++
		counterRespMsg += len(response)

		// 3. объединяем ответ по батчу и мапу [orderUID]->message в структуру
		//    и шлём в канал для обработки в processBatchResponse

		batchInfo := &BatchInfo{
			respOfBatch:      response,
			messageByUID:     messageMap,
			lastBatchMessage: batch[len(batch)-1],
		}
		responses <- batchInfo

		if lenBatches%10000 == 0 {
			log.Printf("batchWorker: обработано %d батчей, из %d сообщений, ответов api на запросы %d, ответов api для %d сообщений, сообщений в DLQ %d, за %v c.\n",
				batchCounter, lenBatches, respCounter, counterRespMsg, counterDLQ, time.Since(start).Seconds())
		}
	}

	log.Printf("batchWorker: канал batches закрыт, обработано %d батчей, из %d сообщений, ответов api на запросы %d, ответов api для %d сообщений, сообщений в DLQ %d, время работы этапа %v c.\n",
		batchCounter, lenBatches, respCounter, counterRespMsg, counterDLQ, time.Since(start).Seconds())
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
func sendToDLQ(w *kafka.Writer, msg *kafka.Message, reason string) {

	if msg == nil {
		log.Println("sendToDLQ: передан nil указатель на сообщение")
		return
	}

	keyStr := string(msg.Key)
	if msg.Key == nil {
		keyStr = "<nil-key>"
	}

	dlqMsg := kafka.Message{
		Key:   []byte(fmt.Sprintf("dlq-%s", keyStr)),
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
		log.Printf("ошибка отправки сообщения %s в DLQ: %v", keyStr, err)
	}
}

// sendBatchWithRetry передает сообщение в API с повторами и получет []OrderResponse в ответ
func sendBatchWithRetry(client *http.Client, data []json.RawMessage) ([]OrderResponse, error) {

	// определяем адрес отправки
	apiURL := fmt.Sprintf("http://%s:%d/order", cfg.ServiceHost, cfg.ServicePort)

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

// processBatchResponse обрабатывает полученные от api ответы по каждому сообщению из батча
func processBatchResponse(r *kafka.Reader, dlqWriter *kafka.Writer, responses <-chan *BatchInfo, endCh chan struct{}, wgPipe *sync.WaitGroup) {

	defer wgPipe.Done()

	// закрываем при выходе канал endCh, чтобы разблокировать main()
	defer func() {
		close(endCh)
		log.Println("processBatchResponse: закрыл кнал endCh.")
	}()

	start := time.Now()

	batchApi := 0     // счётчик поступивших ответов от api
	msgApiAnswer := 0 // количество сообщений, на которые api дало ответ
	toDLQ := 0        // счётчик сообщений, отправленных в DLQ

	// слушаем канал с информацией об ответах api, пока канал открыт
	for batchInfo := range responses {

		batchApi++
		msgApiAnswer += len(batchInfo.respOfBatch)
		if msgApiAnswer%10000 == 0 {
			log.Printf("processBatchResponse: от api поступило ответов %d для %d сообщений, отправлено в DLQ: %d, за %v c.\n",
				batchApi, msgApiAnswer, toDLQ, time.Since(start).Seconds())
		}

		// для обработки ответа по каждому сообщению смотрим есть ли
		// сообщение в мапе и можно ли его закоммитить
		for _, resp := range batchInfo.respOfBatch {

			msg, ok := batchInfo.messageByUID[resp.OrderUID]
			if !ok {
				// не смотря на такое невероятное стечение обстоятельств, чтобы не останавливать конвейер, логируем и продолжаем
				log.Printf("processBatchResponse: в мапе заказов не оказалось сообщения с OrderUID = %s !!!\n", resp.OrderUID)
				continue
			}

			// в DLQ отправляем не только явные ошибки, но и дубликаты сообщений (для выявления возможных причин появления)
			if resp.Status == "badRequest" || resp.Status == "error" || resp.Status == "conflict" {
				sendToDLQ(dlqWriter, msg, resp.Message)
				toDLQ++
			}
		}

		// коммитим один раз весь батч по последнему сообщению в батче
		if err := r.CommitMessages(context.Background(), *batchInfo.lastBatchMessage); err != nil {
			log.Printf("processBatchResponse: ошибка коммита батча по сообщению %s: %v", string(batchInfo.lastBatchMessage.Key), err)
			// TODO возможно следует добавить логику смещения к предыдущему сообщению или просто любому сообщению в батче, чтобы закоммитить хоть что-то
		}
	}

	log.Printf("processBatchResponse: от api поступило ответов %d для %d сообщений, отправлено в DLQ: %d, время работы этапа %v c.\n",
		batchApi, msgApiAnswer, toDLQ, time.Since(start).Seconds())
}

func main() {

	// считываем конфигурацию
	cfg = readConfig()

	// заведём контекст для отмены работы консумера
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// обработка сигналов ОС для graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// канал для передачи ошибки ридера при чтении сообщений из кафки
	errCh := make(chan error)

	// канал для передачи сигнала об окончании обработки сообщений
	endCh := make(chan struct{})

	// ждём сигнал отмены в фоне
	go func() {
		<-sigChan
		log.Println("Получен сигнал остановки, завершаем работу...")
		cancel()
	}()

	// запускаем основной код консумера
	go consumer(ctx, errCh, endCh)

	// ждём получения ошибки или nil из логики конвейера
	// ошибки: нет возможности читать сообщения из брокера или нет возможности заполнять DLQ
	err := <-errCh
	if err != nil {
		log.Printf("консумер завершился с критической ошибкой: %v", err)
		cancel()
	}

	<-endCh // завершился последний воркер в конвейере обработки

	if err == nil {
		log.Println("Консумер корректно завершил работу.")
	}
}
