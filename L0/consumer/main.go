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

	// для трейсинга
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	// для Prometheus метрик
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// провайдер трейсов для продюсера
var tracer trace.Tracer

// initTracing - инициализация трейсинга
func initTracing() (*sdktrace.TracerProvider, error) {

	ctx := context.Background()

	// экспорт трейсов в jaeger через otlp/grpc
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint("jaeger:4317"), otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("не удалось создать экспортер трейсов: %w.\n", err)
	}

	// создаем провайдер трейсов
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("consumer"),
			semconv.ServiceVersion("1.0.0"),
		)),
	)

	// устанавливаем глобальный провайдер
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	log.Println("Трейсинг консумера инициализирован (jaeger:4317).")
	return tp, nil
}

// прометеус метрики для консумера
var (
	// для RPS по этапам - считаем сообщения на каждом этапе
	consumerStageMessages = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "consumer_stage_messages_total",
		Help: "Количество сообщений на каждом этапе конвейера",
	}, []string{"stage"}) // stage: read, batched, prepared, sent, processed, dlq

	// для времени ответа API
	consumerApiResponseTime = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "consumer_api_response_duration_seconds",
		Help:    "Время ответа API на батч запросов",
		Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5},
	})

	// для lag консумера (гистограмма для среднего/макс)
	consumerLagHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "consumer_lag_seconds",
		Help:    "Задержка обработки сообщений",
		Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
	})
	// Среднее: avg(rate(consumer_lag_seconds_sum[5m])) / avg(rate(consumer_lag_seconds_count[5m]))
	// Максимальное: max_over_time(histogram_quantile(0.99, rate(consumer_lag_seconds_bucket[5m]))[5m:1m])
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
	countClientConst    = 10             // количество отправителей запросов по батчам в api по умолчанию
	clientTimeoutConst  = 30             // таймаут для HTTP клиента по умолчанию
	dlqTopicConst       = "my-topic-DLQ" // топик для DLQ
)

// MessageWithTrace оборачивает kafka.Message вместе с его контекстом трейсинга для передачи trace через этапы пайплайна
type MessageWithTrace struct {
	Message   *kafka.Message  // само сообщение
	Ctx       context.Context // контекст, содержащий span этого сообщения
	Span      trace.Span      // сам span сообщения
	StartTime time.Time       // для метрики времени обработки
}

// MessageForAPI структура для передачи трейса в api
type MessageForAPI struct {
	Data        json.RawMessage `json:"data"`
	Traceparent string          `json:"traceparent,omitempty"`
}

// PrepareBatch структура для параллельной отправки ограниченного величиной COUNT_CLIENT количества батчей в api
type PrepareBatch struct {
	batchMessages    []MessageForAPI              // указатели на подготовленные к отправке в api сообщения с трейсами
	messageByUID     map[string]*MessageWithTrace // мапа идентификации [orderUID]->MessageWithTrace
	lastBatchMessage *kafka.Message               // указатель на последнее сообщение в батче, чтобы коммитить весь батч, так как порядок сообщений в батче сохраняется
}

// RespBatchInfo объединяет информацию об ответах api по сообщениям с самими сообщениями
type RespBatchInfo struct {
	respOfBatch      []OrderResponse              // ответы по каждому из сообщений батча
	messageByUID     map[string]*MessageWithTrace // мапа идентификации [orderUID]->MessageWithTrace
	lastBatchMessage *kafka.Message               // указатель на последнее сообщение в батче, чтобы коммитить весь батч, так как порядок сообщений в батче сохраняется
}

// OrderResponse структура для ответов из api (копия из postOrder.go)
type OrderResponse struct {
	OrderUID   string `json:"order_uid"`         // идентификатор сообщения
	Status     string `json:"status"`            // статус: "success", "conflict", "badRequest", "error"
	MessageErr string `json:"message,omitempty"` // информация об ошибке
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
	CountClient    int           // количество отправителей запросов по батчам в api
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
		ServicePort:    getEnvInt("SERVICE_PORT", servicePortConst),
		BatchSize:      getEnvInt("BATCH_SIZE_NUM", batchSizeConst),
		BatchTimeout:   time.Duration(getEnvInt("BATCH_TIMEOUT_S", batchTimeoutConst)) * time.Second,
		MaxRetries:     getEnvInt("MAX_RETRIES_NUM", maxRetriesConst),
		RetryDelayBase: time.Duration(getEnvInt("RETRY_DELEY_BASE_MS", retryDelayBaseConst)) * time.Millisecond,
		CountClient:    getEnvInt("COUNT_CLIENT", countClientConst),
		ClientTimeout:  time.Duration(getEnvInt("CLIENT_TIMEOUT_S", clientTimeoutConst)) * time.Second,
		DlqTopic:       getEnvString("DLQ_TOPIC_NAME_STR", dlqTopicConst),
	}
}

// consumer это основной код консумера
func consumer(ctx context.Context, errCh chan<- error, endCh chan struct{}) {

	// уточняем контекст для трейсинга
	ctx, span := tracer.Start(ctx, "consumer.pipeline.total")
	defer span.End()
	span.SetAttributes(
		attribute.String("kafka.topic", cfg.Topic),
		attribute.String("kafka.group.id", cfg.GroupID),
	)

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
		// MaxWait:  cfg.BatchTimeout, // FetchMessage не учитывает
	})
	defer func() {
		if err := r.Close(); err != nil {
			log.Printf("ошибка при закрытии ридера Kafka: %v", err)
		}
	}()

	log.Printf("Консумер подписан на топик '%s' в группе '%s'.\n", r.Config().Topic, r.Config().GroupID)
	log.Printf("DLQ writer консумера подписан на топик '%s'.\n", dlqWriter.Topic)
	log.Println("Начинаем вычитывать !!!")

	// размер буферов каналов следует назначать исходя из сетевых задержек и ожидаемой пропускной
	// способности пайплайна. Интересно: есть ли какая-то формула или практический подход?
	// Слишком большой размер буферов приводит к длительному grace периоду при остановке контейнера.
	messagesCh := make(chan *MessageWithTrace, cfg.BatchSize*10) // канал для входящих сообщений с большим буфером
	batchesCh := make(chan []*MessageWithTrace, cfg.BatchSize/4) // канал для передачи батчей на обработку
	preparesCh := make(chan *PrepareBatch, cfg.BatchSize/4)      // канал для передачи подготовленной информации к отправке в api
	collectCh := make(chan []*PrepareBatch, cfg.BatchSize/4)     // канал скомпанованных данных о батчах для параллельной передачи в api
	responsesCh := make(chan *RespBatchInfo, cfg.BatchSize/4)    // канал передачи ответов по батчам и мап с сообщениями

	// wgPipe для ожидания всех горутин конвейера
	var wgPipe sync.WaitGroup

	// 1. читаем сообщения из кафки
	wgPipe.Add(1)
	go readMsgOfKafka(ctx, r, messagesCh, errCh, &wgPipe)

	// 2. комплектуем батчи из прочитанных сообщений
	wgPipe.Add(1)
	go complectBatches(messagesCh, batchesCh, &wgPipe)

	// 3. подготавливаем информацию для отправки
	wgPipe.Add(1)
	go prepareBatchToSending(dlqWriter, batchesCh, preparesCh, &wgPipe)

	// 4. собираем данные по батчам в группы для параллельной отправки в api
	wgPipe.Add(1)
	go batchPrepareCollect(preparesCh, collectCh, &wgPipe)

	// 5. параллельно направляем запросы в api по группе батчей
	wgPipe.Add(1)
	go sendBatchInfo(dlqWriter, collectCh, responsesCh, &wgPipe)

	// 6. обрабатываем ответы api для каждого сообщения - коммитим или заполняем DLQ
	wgPipe.Add(1)
	go processBatchResponse(r, dlqWriter, responsesCh, endCh, &wgPipe)

	// ждём окончания обработки всех считанных readMsgOfKafka сообщений
	wgPipe.Wait()

	// close(errCh) уже выполнился при выходе из readMsgOfKafka
	// close(endCh) уже выполнился при выходе из processBatchResponse
}

// extractTraceFromHeaders извлекает контекст трейсинга из заголовков Kafka
func extractTraceFromHeaders(ctx context.Context, headers []kafka.Header) context.Context {

	if len(headers) == 0 {
		return ctx
	}

	carrier := propagation.HeaderCarrier{}
	for _, h := range headers {
		if h.Key == "traceparent" {
			carrier.Set("traceparent", string(h.Value))
			break
		}
	}

	if carrier.Get("traceparent") != "" {
		return otel.GetTextMapPropagator().Extract(ctx, carrier)
	}

	return ctx
}

// readMsgOfKafka читает сообщения из кафки и наполняет канал messagesCh
func readMsgOfKafka(ctx context.Context, r *kafka.Reader, messagesCh chan<- *MessageWithTrace, errCh chan<- error, wgPipe *sync.WaitGroup) {

	start := time.Now()

	defer wgPipe.Done()

	defer func() {
		// закрываем при выходе канал messagesCh, чтобы по мере обработки
		// сообщений из канала завершили работу последующие этапы обработки
		close(messagesCh)
		log.Println("readMsgOfKafka: закрыл канал messagesCh.")

		// закрываем при выходе канал errCh, чтобы разблокировать main()
		close(errCh)
		log.Println("readMsgOfKafka: закрыл канал errCh.")
	}()

	inMsgCounter := 0  // счётчик входящих сообщений для логирования
	outMsgCounter := 0 // счётчик отправленных в канал сообщений

	for {

		msg, err := r.FetchMessage(ctx)
		if err != nil {
			// если контекст отменили (graceful shutdown)
			if errors.Is(err, context.Canceled) {
				log.Printf("readMsgOfKafka: чтение из kafka завершено, получено %d сообщений, отправлено на батчирование %d сообщений, за %v с.\n",
					inMsgCounter, outMsgCounter, time.Since(start).Seconds())
				errCh <- nil // оповещаем main() и выходим, конвейер продолжает обработку уже вычитанных сообщений
				return
			}
			log.Printf("readMsgOfKafka: чтение из kafka завершено, получено %d сообщений, отправлено на батчирование %d сообщений, за %v с.\n",
				inMsgCounter, outMsgCounter, time.Since(start).Seconds())
			errCh <- err // оповещаем main() и выходим, конвейер продолжает обработку уже вычитанных сообщений
			return
		}

		// извлекаем трейс из заголовков
		msgCtx := extractTraceFromHeaders(ctx, msg.Headers)

		// создаем span для этапа "чтение из Kafka" для конкретного сообщения,
		// он будет дочерним для span из функции consumer
		msgCtx, span := tracer.Start(msgCtx, "consumer.pipeline.stage",
			trace.WithAttributes(
				attribute.String("stage", "kafka_read"),
				attribute.String("message.key", string(msg.Key)),
				attribute.String("order.uid", extractOrderUID(msg.Value)),
				attribute.Int("kafka.partition", msg.Partition),
				attribute.Int64("kafka.offset", msg.Offset),
				attribute.Int("message.number", inMsgCounter),
			))

		// считаем lag для мониторинга
		if !msg.Time.IsZero() {
			lag := time.Since(msg.Time).Seconds()
			consumerLagHistogram.Observe(lag)
		}

		// метрика RPS для этапа
		consumerStageMessages.WithLabelValues("read").Inc()

		// заворачиваем сообщение и контекст в структуру
		wrappedMsg := &MessageWithTrace{
			Message:   &msg,
			Ctx:       msgCtx,
			Span:      span,
			StartTime: time.Now(), // фиксируем время начала обработки
		}

		inMsgCounter++ // добавляем входящий счётчик
		// отправляем обёртку в канал (сам span завершаем позже в processBatchResponse)
		messagesCh <- wrappedMsg
		outMsgCounter++ // добавляем исходящий счётчик

		if inMsgCounter%10000 == 0 {
			log.Printf("readMsgOfKafka: получено %d сообщений, отправлено на батчирование %d сообщений, за %v с.\n",
				inMsgCounter, outMsgCounter, time.Since(start).Seconds())
		}
	}
}

// complectBatches собирает батчи из сообщений из messagesCh (по количеству или по времени) и наполняет канал batches,
// контекст не используем в надежде обработать все уже поступившие в канал messagesCh сообщения
func complectBatches(messagesCh <-chan *MessageWithTrace, batchesCh chan<- []*MessageWithTrace, wgPipe *sync.WaitGroup) {

	defer wgPipe.Done()

	// закрываем при выходе канал batchesCh, чтобы по мере обработки
	// сообщений из канала завершили работу последующие этапы обработки
	defer func() {
		close(batchesCh)
		log.Println("complectBatches: закрыл канал batchesCh.")
	}()

	start := time.Now()

	currentBatch := make([]*MessageWithTrace, 0, cfg.BatchSize) // батч с указателями на сообщения в обёртке мониторинга
	ticker := time.NewTicker(cfg.BatchTimeout)                  // таймер для отключения комплектования батча по времени
	defer ticker.Stop()

	inMsgCounter := 0    // счётчик входящих сообщений
	outMsgCounter := 0   // счётчик количества сообщений в отправленных батчах
	outBatchCounter := 0 // счётчик количества отправленных батчей

	sendBatch := func() {
		// создаем копию батча
		copyBatch := make([]*MessageWithTrace, len(currentBatch))
		copy(copyBatch, currentBatch)
		// отправляем батч в канал
		batchesCh <- copyBatch
		// метрика: сообщения на этапе batched
		consumerStageMessages.WithLabelValues("batched").Add(float64(len(copyBatch)))

		// очищаем память и плюсуем счетчики
		currentBatch = currentBatch[:0:cfg.BatchSize]
		outMsgCounter += len(copyBatch)
		outBatchCounter++
	}

	for {
		select {
		// если прошло время, выделенное на сбор батча, отправляем что накопилось (если накопилось) и повторяем вычитывание
		case <-ticker.C:
			if len(currentBatch) > 0 {
				sendBatch()
			}
			continue
		// если можем считать сообщение и канал открыт, читаем сообщение и копим батч
		case msg, ok := <-messagesCh:
			if !ok { // если канал закрыт и читать больше нечего, отправляем что накопилось (если накопилось) и завершаем работу
				if len(currentBatch) > 0 {
					sendBatch()
				}
				log.Printf("complectBatches: канал messagesCh закрыт, получено %d сообщений, отправлено %d сообщений в %d батчах, время работы этапа %v c.\n",
					inMsgCounter, outMsgCounter, outBatchCounter, time.Since(start).Seconds())
				return
			}
			inMsgCounter++ // добавляем входящий счётчик

			// создаем span для этапа батчирования этого сообщения
			batchCtx, _ := tracer.Start(msg.Ctx, "consumer.pipeline.stage",
				trace.WithAttributes(
					attribute.String("stage", "batching"),
					attribute.Int("batch.current_size", len(currentBatch)+1),
					attribute.Int("message.index_in_batch", len(currentBatch)),
				))

			// обновляем контекст сообщения
			msg.Ctx = batchCtx

			currentBatch = append(currentBatch, msg) // дополняем батч
			// если достигли нужного размера сразу отправляем
			if len(currentBatch) >= cfg.BatchSize {
				sendBatch()
			}

			if inMsgCounter%10000 == 0 {
				log.Printf("complectBatches: получено %d сообщений, отправлено %d сообщений в %d батчах, за %v c.\n",
					inMsgCounter, outMsgCounter, outBatchCounter, time.Since(start).Seconds())
			}
		}
	}
}

// extractTraceparent извлекает traceparent из контекста
func extractTraceparent(ctx context.Context) string {

	carrier := propagation.HeaderCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	return carrier.Get("traceparent")
}

// prepareBatchToSending подготавливает информацию по батчу к отправке в api
func prepareBatchToSending(dlqWriter *kafka.Writer, batchesCh <-chan []*MessageWithTrace, preparesCh chan<- *PrepareBatch, wgPipe *sync.WaitGroup) {

	defer wgPipe.Done()

	// закрываем при выходе канал preparesCh, чтобы по мере обработки
	// сообщений из канала завершили работу последующие этапы обработки
	defer func() {
		close(preparesCh)
		log.Println("prepareBatchToSending: закрыл канал preparesCh.")
	}()

	start := time.Now()

	inBatchCounter := 0     // счётчик пришедших батчей
	inMsgCounter := 0       // счётчик обработанных сообщений
	msgInDLQ := 0           // количество сообщений, отправленных в DLQ
	outPackInfoCounter := 0 // количество подготовленных к отправке в api батчей

	// слушаем канал с группами сообщений
	for batch := range batchesCh {

		// нулевых батчей приходить не должно, но на всякий случай проверяем
		if len(batch) == 0 {
			log.Printf("prepareBatchToSending: следующим после батча %d пришёл батч с нулевой длинной.", inBatchCounter)
			continue
		}

		inBatchCounter++           // подсчитываем пришедшие батчи
		inMsgCounter += len(batch) // подсчитываем пришедшие сообщения

		// подготавливаем данные для API
		batchMessages := make([]MessageForAPI, 0, len(batch))
		messageMap := make(map[string]*MessageWithTrace, len(batch)) // мапа идентификации [orderUID]->*MessageWithTrace

		for i := range batch {

			// создаем span для этапа подготовки этого сообщения
			prepareCtx, _ := tracer.Start(batch[i].Ctx, "consumer.pipeline.stage",
				trace.WithAttributes(
					attribute.String("stage", "preparing"),
					attribute.Int("batch_index", inBatchCounter),
					attribute.Int("message_index_in_batch", i),
				))

			// обновляем контекст сообщения
			batch[i].Ctx = prepareCtx

			// извлекаем orderUID для маппинга
			if orderUID := extractOrderUID(batch[i].Message.Value); orderUID != "" {
				messageMap[orderUID] = batch[i]
				// создаем MessageForAPI с traceparent
				batchMessages = append(batchMessages, MessageForAPI{
					Data:        batch[i].Message.Value,
					Traceparent: extractTraceparent(batch[i].Ctx), // извлекаем traceparent
				})
			} else {
				// при отсутствии идентификатора сообщение шлём в DLQ
				msgInDLQ++
				sendToDLQ(dlqWriter, batch[i].Message, "not exist OrderUID")
			}
		}

		// метрика для этапа подготовки
		consumerStageMessages.WithLabelValues("prepared").Add(float64(len(batchMessages)))

		// результат подготовки упаковываем в структуру
		prepareBatch := &PrepareBatch{
			batchMessages:    batchMessages,
			messageByUID:     messageMap,
			lastBatchMessage: batch[len(batch)-1].Message,
		}

		preparesCh <- prepareBatch // отправляем пакет с данными
		outPackInfoCounter++       // подсчитываем отправленные пакеты данных

		if inMsgCounter%10000 == 0 {
			log.Printf("prepareBatchToSending: обработано %d батчей, из %d сообщений, на отправку в api передано %d батчей, сообщений в DLQ %d, за %v c.\n",
				inBatchCounter, inMsgCounter, outPackInfoCounter, msgInDLQ, time.Since(start).Seconds())
		}
	}

	log.Printf("prepareBatchToSending: канал preparesCh закрыт, обработано %d батчей, из %d сообщений, на отправку в api передано %d батчей, сообщений в DLQ %d, время работы этапа %v c.\n",
		inBatchCounter, inMsgCounter, outPackInfoCounter, msgInDLQ, time.Since(start).Seconds())
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

// batchPrepareCollect получает пакеты данных о батчах, формирует их по COUNT_CLIENT штук и направляет на параллельную передачу в api
func batchPrepareCollect(preparesCh <-chan *PrepareBatch, collectCh chan<- []*PrepareBatch, wgPipe *sync.WaitGroup) {

	defer wgPipe.Done()

	// закрываем при выходе канал collectCh, чтобы по мере обработки
	// сообщений из канала завершили работу последующие этапы обработки
	defer func() {
		close(collectCh)
		log.Println("batchPrepareCollect: закрыл канал collectCh.")
	}()

	start := time.Now()

	currentCollect := make([]*PrepareBatch, 0, cfg.CountClient) // группа с батчами для отправки в api
	ticker := time.NewTicker(cfg.BatchTimeout)                  // используем таймер для отключения комплектования батча по времени
	defer ticker.Stop()

	inMsgCounter := 0        // счётчик сообщений, прошедших через batchPrepareCollect
	inBatchInfoCounter := 0  // счётчик количества батчей, о которых получена информация
	outBatchInfoCounter := 0 // счётчик батчей в группах
	outCollectCounter := 0   // счётчик количества получившихся групп

	sendCollect := func() {
		copyBatch := make([]*PrepareBatch, len(currentCollect))
		copy(copyBatch, currentCollect)
		collectCh <- copyBatch

		// метрика для этапа сбора
		consumerStageMessages.WithLabelValues("collected").Add(float64(len(currentCollect)))

		currentCollect = currentCollect[:0:cfg.CountClient]
		outBatchInfoCounter += len(copyBatch)
		outCollectCounter++
	}

	for {
		select {
		// если прошло время, принятое для сбора группы, отправляем что накопилось (если накопилось) и повторяем вычитывание
		case <-ticker.C:
			if len(currentCollect) > 0 {
				sendCollect()
			}
			continue
		// если можем считать сообщение и канал открыт, читаем сообщение и копим группу
		case prepareBatch, ok := <-preparesCh:
			if !ok { // если канал закрыт и читать больше нечего, отправляем что накопилось (если накопилось) и завершаем работу
				if len(currentCollect) > 0 {
					sendCollect()
				}
				log.Printf("batchPrepareCollect: канал preparesCh закрыт, получено %d сообщений в %d батчах, отправлено %d батчей в %d группах, время работы этапа %v c.\n",
					inMsgCounter, inBatchInfoCounter, outBatchInfoCounter, outCollectCounter, time.Since(start).Seconds())
				return
			}

			inBatchInfoCounter++
			inMsgCounter += len(prepareBatch.batchMessages)

			// создаем span для этапа сбора для каждого сообщения в батче
			for _, wrappedMsg := range prepareBatch.messageByUID {
				if wrappedMsg != nil { // проверка на случай нахождения сообщения в dlq
					collectCtx, _ := tracer.Start(wrappedMsg.Ctx, "consumer.pipeline.stage",
						trace.WithAttributes(
							attribute.String("stage", "collecting"),
							attribute.Int("collect_group_size", len(currentCollect)+1),
						))
					wrappedMsg.Ctx = collectCtx
				}
			}

			currentCollect = append(currentCollect, prepareBatch) // дополняем группу
			// если достигли нужного размера сразу отправляем
			if len(currentCollect) >= cfg.CountClient {
				sendCollect()
			}

			if inMsgCounter%10000 == 0 {
				log.Printf("batchPrepareCollect: получено %d сообщений в %d батчах, отправлено %d батчей в %d группах, за %v c.\n",
					inMsgCounter, inBatchInfoCounter, outBatchInfoCounter, outCollectCounter, time.Since(start).Seconds())
			}
		}
	}
}

// sendBatchInfo передает сообщение (батч) в API с повторами и получет ответ по каждому сообщению ([]OrderResponse)
func sendBatchInfo(dlqWriter *kafka.Writer, collectCh <-chan []*PrepareBatch, responsesCh chan<- *RespBatchInfo, wgPipe *sync.WaitGroup) {

	defer wgPipe.Done()

	// закрываем при выходе канал responsesCh, чтобы по мере обработки
	// сообщений из канала завершили работу последующие этапы обработки
	defer func() {
		close(responsesCh)
		log.Println("sendBatchInfo: закрыл канал responsesCh.")
	}()

	start := time.Now()

	var (
		batchInfoCounter int64 // количество батчей, прошедших через sendBatchInfo
		counterMsg       int64 // количество сообщений, прошедших через sendBatchInfo
		msgInDLQ         int64 // количество сообщений, отправленных в DLQ
		inRespCounter    int64 // количество ответов по запросам
		inRespMsgCounter int64 // количество ответов по сообщениям
	)

	// клиент для отправки вычитанных из кафки сообщений на api сервиса
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: cfg.CountClient, // cколько одновременных соединений держать открытыми
			DisableKeepAlives:   false,           // соединения переиспользуются, не создаются новые каждый раз
		},
		Timeout: cfg.ClientTimeout,
	}

	// определяем адрес отправки запросов
	apiURL := fmt.Sprintf("http://%s:%d/order", cfg.ServiceHost, cfg.ServicePort)

	// вычитываем из канала очередной слайс с информацией о батчах
	for packInfo := range collectCh {

		var wgSend sync.WaitGroup

		// запускаем пулл воркеров для отправки запросов по батчам в api
		for i := 0; i < len(packInfo); i++ {
			wgSend.Add(1)
			go func(idx int) {
				defer wgSend.Done()

				atomic.AddInt64(&batchInfoCounter, 1)
				atomic.AddInt64(&counterMsg, int64(len(packInfo[idx].batchMessages)))

				for _, wrappedMsg := range packInfo[idx].messageByUID {
					if wrappedMsg != nil {
						sendCtx, _ := tracer.Start(wrappedMsg.Ctx, "consumer.pipeline.stage",
							trace.WithAttributes(
								attribute.String("stage", "sending"),
								attribute.String("api.url", apiURL),
							))
						wrappedMsg.Ctx = sendCtx // обновляем контекст
					}
				}

				// метрика этапа отправки
				consumerStageMessages.WithLabelValues("sent").Add(float64(len(packInfo[idx].batchMessages)))

				// сериализуем batchMessages (уже содержат traceparent)
				requestBody, err := json.Marshal(packInfo[idx].batchMessages)
				if err != nil {
					log.Printf("ошибка сериализации: %v\n", err)
					for _, wrappedMsg := range packInfo[idx].messageByUID {
						atomic.AddInt64(&msgInDLQ, 1)
						sendToDLQ(dlqWriter, wrappedMsg.Message, err.Error())
					}
					return
				}

				// берём контекст первого сообщения для создания HTTP запроса
				var traceCtx context.Context
				for _, wrappedMsg := range packInfo[idx].messageByUID {
					traceCtx = wrappedMsg.Ctx
					break
				}
				if traceCtx == nil {
					traceCtx = context.Background()
				}

				// создаём span для HTTP запроса
				httpCtx, httpSpan := tracer.Start(traceCtx, "consumer.http.request",
					trace.WithAttributes(
						attribute.String("api.url", apiURL),
						attribute.Int("batch.size", len(packInfo[idx].batchMessages)),
					))
				defer httpSpan.End()

				// с повторами отправляем батч в api
				for attempt := 1; attempt <= cfg.MaxRetries; attempt++ {

					requestStart := time.Now() // засекаем время

					// создаём запрос с контекстом
					req, err := http.NewRequestWithContext(httpCtx, "POST", apiURL, bytes.NewReader(requestBody))
					if err != nil {
						log.Printf("Ошибка создания запроса (попытка %d/%d): %v.\n", attempt, cfg.MaxRetries, err)
						continue
					}
					req.Header.Set("Content-Type", "application/json")

					// инжектируем трейс из httpCtx в заголовки
					otel.GetTextMapPropagator().Inject(httpCtx, propagation.HeaderCarrier(req.Header))

					// направляем запрос
					resp, err := client.Do(req)

					// метрика времени ответа
					if err == nil && resp != nil {
						duration := time.Since(requestStart).Seconds()
						consumerApiResponseTime.Observe(duration)
					}

					if err != nil {
						log.Printf("Ошибка сети (попытка %d/%d): %v.\n", attempt, cfg.MaxRetries, err)
						if attempt < cfg.MaxRetries {
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

						var orderResponses []OrderResponse
						if err := json.Unmarshal(body, &orderResponses); err != nil {
							// если ответ не парсится => батч в DLQ
							for _, wrappedMsg := range packInfo[idx].messageByUID {
								atomic.AddInt64(&msgInDLQ, 1)
								sendToDLQ(dlqWriter, wrappedMsg.Message, err.Error())
							}
							log.Println("sendBatchInfo: от api получен неожиданный ответ - батч направлен в DLQ.")
							return
						}

						// считаем статистику
						atomic.AddInt64(&inRespCounter, 1)
						atomic.AddInt64(&inRespMsgCounter, int64(len(orderResponses)))

						// обновляем span HTTP запроса
						httpSpan.SetAttributes(
							attribute.Int("http.status_code", resp.StatusCode),
							attribute.Int("attempt", attempt),
						)
						httpSpan.SetStatus(codes.Ok, "success")

						// объединяем ответ по батчу и мапу [orderUID]->MessageWithTrace в структуру
						// и шлём в канал для обработки в processBatchResponse
						respBatchInfo := &RespBatchInfo{
							respOfBatch:      orderResponses,
							messageByUID:     packInfo[idx].messageByUID,
							lastBatchMessage: packInfo[idx].lastBatchMessage,
						}
						responsesCh <- respBatchInfo

						if counterMsg%10000 == 0 {
							log.Printf("sendBatchInfo: обработано %d батчей, из %d сообщений, ответов api на запросы %d, ответов api для %d сообщений, сообщений в DLQ %d, за %v c.\n",
								atomic.LoadInt64(&batchInfoCounter), atomic.LoadInt64(&counterMsg), atomic.LoadInt64(&inRespCounter),
								atomic.LoadInt64(&inRespMsgCounter), atomic.LoadInt64(&msgInDLQ), time.Since(start).Seconds())
						}
						// если ответ поступил и статус корректен - завершаем горутину
						return
					}

					// если статус не 201 или 207, пробуем снова
					log.Printf("Попытка %d/%d: неожиданный статус %d", attempt, cfg.MaxRetries, resp.StatusCode)
				}

				// если за повторы не получилось отправить запрос в api, то отправляем всё в DLQ
				for _, wrappedMsg := range packInfo[idx].messageByUID {
					atomic.AddInt64(&msgInDLQ, 1)
					sendToDLQ(dlqWriter, wrappedMsg.Message, "max retries exceeded")
				}
				httpSpan.SetStatus(codes.Error, "all retries failed")
			}(i)
		}

		// ждём окончания работы по всей группе батчей
		wgSend.Wait()
	}

	log.Printf("sendBatchInfo: канал collectCh закрыт, обработано %d батчей, из %d сообщений, ответов api на запросы %d, ответов api для %d сообщений, сообщений в DLQ %d, время работы этапа %v c.\n",
		batchInfoCounter, counterMsg, inRespCounter, inRespMsgCounter, msgInDLQ, time.Since(start).Seconds())
}

// processBatchResponse обрабатывает полученные от api ответы по каждому сообщению из батча
func processBatchResponse(r *kafka.Reader, dlqWriter *kafka.Writer, responsesCh <-chan *RespBatchInfo, endCh chan struct{}, wgPipe *sync.WaitGroup) {

	defer wgPipe.Done()

	// закрываем при выходе канал endCh, чтобы разблокировать main()
	defer func() {
		close(endCh)
		log.Println("processBatchResponse: закрыл канал endCh.")
	}()

	start := time.Now()

	batchApiAnswer := 0 // счётчик поступивших ответов от api
	msgApiAnswer := 0   // количество сообщений, на которые api дало ответ
	msgInDLQ := 0       // счётчик сообщений, отправленных в DLQ

	// слушаем канал с информацией об ответах api, пока канал открыт
	for batchInfo := range responsesCh {

		batchApiAnswer++
		msgApiAnswer += len(batchInfo.respOfBatch)

		if msgApiAnswer%10000 == 0 {
			log.Printf("processBatchResponse: от api поступило ответов %d для %d сообщений, отправлено в DLQ: %d, за %v c.\n",
				batchApiAnswer, msgApiAnswer, msgInDLQ, time.Since(start).Seconds())
		}

		// для обработки ответа по каждому сообщению смотрим есть ли
		// сообщение в мапе и можно ли его закоммитить
		for _, resp := range batchInfo.respOfBatch {

			wrappedMsg, ok := batchInfo.messageByUID[resp.OrderUID]
			if !ok {
				// не смотря на такое невероятное стечение обстоятельств, чтобы не останавливать конвейер, логируем и продолжаем
				log.Printf("processBatchResponse: в мапе заказов не оказалось сообщения с OrderUID = %s !!!\n", resp.OrderUID)
				continue
			}

			// создаем span для этапа обработки ответа
			_, processSpan := tracer.Start(wrappedMsg.Ctx, "consumer.pipeline.stage",
				trace.WithAttributes(
					attribute.String("stage", "processing"),
					attribute.String("api.response.status", resp.Status),
				))

			// в DLQ отправляем не только явные ошибки, но и дубликаты сообщений (для выявления возможных причин появления)
			if resp.Status == "badRequest" || resp.Status == "error" || resp.Status == "conflict" {
				sendToDLQ(dlqWriter, wrappedMsg.Message, resp.MessageErr)
				msgInDLQ++
			}

			// метрика для обработанных сообщений
			consumerStageMessages.WithLabelValues("processed").Inc()

			processSpan.End()

			// завершаем основной span сообщения (завершатся все дочерние spans)
			wrappedMsg.Span.End()
		}

		// коммитим один раз весь батч по последнему сообщению в батче
		if err := r.CommitMessages(context.Background(), *batchInfo.lastBatchMessage); err != nil {
			log.Printf("processBatchResponse: ошибка коммита батча по сообщению %s: %v", string(batchInfo.lastBatchMessage.Key), err)
			// TODO возможно следует добавить логику смещения к предыдущему сообщению или просто любому сообщению в батче, чтобы закоммитить хоть что-то
		}
	}

	log.Printf("processBatchResponse: от api поступило ответов %d для %d сообщений, отправлено в DLQ: %d, время работы этапа %v c.\n",
		batchApiAnswer, msgApiAnswer, msgInDLQ, time.Since(start).Seconds())
}

func main() {

	// запускаем сервер для метрик
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		port := ":8889"
		log.Printf("Prometheus метрики доступны на http://localhost%s/metrics\n", port)
		if err := http.ListenAndServe(port, nil); err != nil && err != http.ErrServerClosed {
			log.Printf("Ошибка запуска сервера метрик: %v.\n", err)
		}
	}()

	// инициализируем трейсинг
	var tp *sdktrace.TracerProvider
	var err error

	tp, err = initTracing()
	if err != nil {
		log.Printf("Не удалось инициализировать трейсинг: %v. Работаем без него.\n", err)
		// создаем noop-трейсер для работы без трассировки
		tracer = noop.NewTracerProvider().Tracer("noop-consumer")
	} else {
		defer func() {
			if err := tp.Shutdown(context.Background()); err != nil {
				log.Printf("Ошибка при остановке провайдера трейсов: %v.\n", err)
			}
		}()
		// получаем реальный трейсер из провайдера
		tracer = tp.Tracer("consumer")
	}

	// считываем конфигурацию
	cfg = readConfig()

	// контекст для отмены работы консумера
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
	// ошибки: нет возможности читать сообщения из брокера
	err = <-errCh
	if err != nil {
		log.Printf("консумер завершился с критической ошибкой: %v", err)
		cancel()
	}

	<-endCh // завершился последний воркер в конвейере обработки

	if err == nil {
		log.Println("Консумер корректно завершил работу.")
	}
}
