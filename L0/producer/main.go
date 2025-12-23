package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/brianvoe/gofakeit/v7"
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
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// выносим константы конфигурации по умолчанию, чтобы были на виду
const (
	topicNameConst     = "my-topic" // имя топика, коррелируется с консумером
	kafkaHostConst     = "kafka"    // имя службы (контейнера) в сети докера по умолчанию
	kafkaPortConst     = 9092       // порт, на котором сидит kafka по умолчанию
	massagesCountConst = 10         // количество сообщений, отправляемых одним врайтером, по умолчанию
	writersCountConst  = 5000       // количество врайтеров для имитации отправки "со всех сторон", по умолчанию
)

// провайдер трейсов для продюсера
var tracer trace.Tracer

// initTracing - инициализация трейсинга для продюсера
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
			semconv.ServiceName("producer"),
			semconv.ServiceVersion("1.0.0"),
		)),
	)

	// устанавливаем глобальный провайдер
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	log.Println("Трейсинг продюсера инициализирован (jaeger:4317).")
	return tp, nil
}

// прометеус метрики для продюсера
var (
	messagesGenerated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "producer_messages_generated_total",
		Help: "Количество сгенерированных сообщений",
	})

	messagesSent = promauto.NewCounter(prometheus.CounterOpts{
		Name: "producer_messages_sent_total",
		Help: "Количество успешно отправленных сообщений",
	})

	sendErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "producer_messages_errors_total",
		Help: "Количество ошибок при отправке",
	})

	generateDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "producer_generate_duration_seconds",
		Help:    "Время генерации сообщений в секундах",
		Buckets: prometheus.DefBuckets, // стандартные бакеты: .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10
	})
)

// ProducerConfig описывает настройки с учётом переменных окружения
type ProducerConfig struct {
	Topic         string // имя топика (коррелируется с консумером)
	KafkaHost     string // имя службы (контейнера) в сети докера
	KafkaPort     int    // порт, на котором сидит kafka
	MassagesCount int    // количество сообщений, отправляемых одним врайтером
	WritersCount  int    // количество врайтеров
}

var cfg *ProducerConfig

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
func readConfig() *ProducerConfig {

	return &ProducerConfig{
		Topic:         getEnvString("TOPIC_NAME_STR", topicNameConst),
		KafkaHost:     getEnvString("KAFKA_HOST_NAME", kafkaHostConst),
		KafkaPort:     getEnvInt("KAFKA_PORT_NUM", kafkaPortConst),
		MassagesCount: getEnvInt("MESSAGES_COUNT", massagesCountConst),
		WritersCount:  getEnvInt("WRITERS_COUNT", writersCountConst),
	}
}

// sendMessages генерирует и отправляет брокеру заданное количество сообщений
func sendMessages(ctx context.Context, w *kafka.Writer, generatedCount, countSended, failedCount *int64, wg *sync.WaitGroup) {

	defer wg.Done()

	// создаем span для отправки сообщений этим врайтером
	sendCtx, sendSpan := tracer.Start(ctx, "producer.send.batch",
		trace.WithAttributes(
			attribute.String("component", "producer"),
			attribute.String("kafka.topic", cfg.Topic),
		))
	defer sendSpan.End()

	// генерируем сообщения для отправки
	messages := messageGenerate(cfg.MassagesCount)
	if len(messages) == 0 {
		log.Println("Не сгенерировано ни одного сообщения для отправки.")
		sendSpan.SetStatus(codes.Error, "нет сообщений для отправки")
		return
	}

	atomic.AddInt64(generatedCount, int64(len(messages)))

	// отправляем сообщения с проверкой контекста
	for i, msgBody := range messages {

		select {
		default:

			// создаем span для отправки одного сообщения
			msgCtx, msgSpan := tracer.Start(sendCtx, "producer.send.message",
				trace.WithAttributes(
					attribute.Int("message.index", i),
					attribute.String("message.key", fmt.Sprintf("Сообщение №%d", i+1)),
					attribute.Int("batch.size", len(messages)),
				))

			// извлекаем order_uid из сообщения для атрибутов
			orderUID := extractOrderUID(msgBody)
			if orderUID != "" {
				msgSpan.SetAttributes(attribute.String("order.uid", orderUID))
			}

			msg := kafka.Message{
				Key:   []byte(fmt.Sprintf("Сообщение №%d", i+1)),
				Value: msgBody,
				Time:  time.Now(),
				// добавляем заголовки для передачи trace_id
				Headers: []kafka.Header{
					{
						Key:   "traceparent",
						Value: []byte(getTraceparentFromContext(msgCtx)),
					},
				},
			}

			err := w.WriteMessages(msgCtx, msg) // используем контекст с трейсом
			if err != nil {
				atomic.AddInt64(failedCount, 1)
				// записываем метрику ошибки
				sendErrors.Inc()
				msgSpan.SetStatus(codes.Error, err.Error())
				msgSpan.SetAttributes(attribute.String("error", err.Error()))
			} else {
				atomic.AddInt64(countSended, 1)
				// записываем метрику успешной отправки
				messagesSent.Inc()
				if *countSended%10000 == 0 {
					log.Printf("Отправлено %d сообщений.\n", *countSended)
				}
				msgSpan.SetStatus(codes.Ok, "сообщение отправлено")
			}

			msgSpan.End()

		case <-ctx.Done():
			sendSpan.SetAttributes(attribute.Bool("cancelled", true))
			return
		}
	}

	sendSpan.SetAttributes(attribute.Int64("sent.count", *countSended), attribute.Int64("failed.count", *failedCount))
}

// getTraceparentFromContext извлекает traceparent из контекста для передачи в кафку
func getTraceparentFromContext(ctx context.Context) string {

	// используем OpenTelemetry пропагатор для получения traceparent
	carrier := propagation.HeaderCarrier{}
	propagation.TraceContext{}.Inject(ctx, carrier)

	traceparent := carrier.Get("traceparent")
	if traceparent == "" {
		// если не удалось извлечь, создаем новый
		span := trace.SpanFromContext(ctx)
		if span != nil {
			sc := span.SpanContext()
			if sc.IsValid() {
				traceparent = fmt.Sprintf("00-%s-%s-01", sc.TraceID(), sc.SpanID())
			}
		}
	}

	return traceparent
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

func main() {

	start := time.Now()

	// инициализируем генератор gofakeit
	gofakeit.Seed(0)

	// запускаем сервер для метрик
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		port := ":8888"
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
		log.Printf("ошибка инициализации трейсинга: %v, работаем без трейсинга.\n", err)
		// создаем noop-трейсер для работы без трассировки
		tracer = noop.NewTracerProvider().Tracer("noop-producer")
	} else {
		defer func() {
			if err := tp.Shutdown(context.Background()); err != nil {
				log.Printf("ошибка остановки трейсинга: %v.\n", err)
			}
		}()
		// получаем реальный трейсер из провайдера
		tracer = tp.Tracer("producer")
	}

	// считываем конфигурацию
	cfg = readConfig()

	// устанавливаем соединение с брокером
	conn, err := kafka.DialLeader(context.Background(), "tcp", fmt.Sprintf("%s:%d", cfg.KafkaHost, cfg.KafkaPort), cfg.Topic, 0)
	if err != nil {
		log.Fatalf("ошибка создания топика кафки: %v.\n", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Ошибка при закрытии продюсером соединения с кафкой: %v.\n", err)
		}
	}()

	log.Println("Соединение с брокером установлено.")

	// создаём ряд врайтеров
	writers := make([]*kafka.Writer, cfg.WritersCount)
	for i := 0; i < len(writers); i++ {
		// определяем продюсер
		writers[i] = &kafka.Writer{
			Addr:         kafka.TCP(fmt.Sprintf("%s:%d", cfg.KafkaHost, cfg.KafkaPort)), // список брокеров
			Topic:        cfg.Topic,                                                     // имя топика, в который будем слать сообщения
			Async:        false,                                                         // можно установить true и получить максимальную скорость без гарантии доставки
			RequiredAcks: kafka.RequireAll,                                              // максимальный контроль доставки (подтверждение от всех реплик)
		}
		defer func() {
			if err := writers[i].Close(); err != nil {
				log.Printf("Ошибка при закрытии продюсера: %v.\n", err)
			}
		}()
	}

	// организуем контекст для корректного завершения писателей
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// подготавливаем обработку сигналов ОС
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// фоном слушаем сигналы отмены и отменяем контекст
	go func() {
		<-sigChan
		log.Println("Получен сигнал остановки, завершаем отправку...")
		cancel()
	}()

	var generatedCount int64 // количество сгенерированных сообщений
	var countSended int64    // количество отправленных сообщений
	var failedCount int64    // количество ошибок при отправке сообщений

	log.Printf("Запускаем %d врайтеров.\n", len(writers))

	// добавляем атрибуты в основной span (если есть)
	if span := trace.SpanFromContext(ctx); span != nil {
		span.SetAttributes(
			attribute.Int("writers.count", len(writers)),
			attribute.Int("batch.size", cfg.MassagesCount),
			attribute.String("kafka.topic", cfg.Topic),
			attribute.String("kafka.host", cfg.KafkaHost),
			attribute.Int("kafka.port", cfg.KafkaPort),
		)
	}

	var wg sync.WaitGroup

	for i := 0; i < len(writers); i++ {
		wg.Add(1)
		go sendMessages(ctx, writers[i], &generatedCount, &countSended, &failedCount, &wg)
	}

	wg.Wait()

	log.Println("Врайтеры завершили отправку сообщений брокеру.")

	log.Printf("Сгенерировано сообщений: %d. Отправлено брокеру сообщений: %d. Ошибок при отправке: %d. Время работы: %v c.\n",
		generatedCount, countSended, failedCount, time.Since(start).Seconds())
}

// Заказ
type Order struct {
	OrderUID          string    `json:"order_uid"`
	TrackNumber       string    `json:"track_number"`
	Entry             string    `json:"entry"`
	Delivery          Delivery  `json:"delivery"`
	Payment           Payment   `json:"payment"`
	Items             []Item    `json:"items"`
	Locale            string    `json:"locale"`
	InternalSignature string    `json:"internal_signature"`
	CustomerID        string    `json:"customer_id"`
	DeliveryService   string    `json:"delivery_service"`
	Shardkey          string    `json:"shardkey"`
	SMID              int       `json:"sm_id"`
	DateCreated       time.Time `json:"date_created"`
	OOFShard          string    `json:"oof_shard"`
}

// Доставка
type Delivery struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Zip     string `json:"zip"`
	City    string `json:"city"`
	Address string `json:"address"`
	Region  string `json:"region"`
	Email   string `json:"email"`
}

// Оплата
type Payment struct {
	Transaction  string  `json:"transaction"`
	RequestID    string  `json:"request_id"`
	Currency     string  `json:"currency"`
	Provider     string  `json:"provider"`
	Amount       float64 `json:"amount"`
	PaymentDT    int64   `json:"payment_dt"`
	Bank         string  `json:"bank"`
	DeliveryCost float64 `json:"delivery_cost"`
	GoodsTotal   float64 `json:"goods_total"`
	CustomFee    float64 `json:"custom_fee"`
}

// Позиция заказа
type Item struct {
	ChrtID      int     `json:"chrt_id"`
	TrackNumber string  `json:"track_number"`
	Price       float64 `json:"price"`
	RID         string  `json:"rid"`
	Name        string  `json:"name"`
	Sale        float64 `json:"sale"`
	Size        string  `json:"size"`
	TotalPrice  float64 `json:"total_price"`
	NMID        int     `json:"nm_id"`
	Brand       string  `json:"brand"`
	Status      int     `json:"status"`
}

// createDelivery выдаёт экземляр структуры Delivery
func createDelivery() Delivery {
	return Delivery{
		Name:    gofakeit.Name(),
		Phone:   gofakeit.Phone(),
		Zip:     gofakeit.Zip(),
		City:    gofakeit.City(),
		Address: gofakeit.Street() + ", " + gofakeit.StreetNumber(),
		Region:  gofakeit.State(),
		Email:   gofakeit.Email(),
	}
}

// createPayment выдаёт экземляр структуры Payment
func createPayment() Payment {
	goodsTotal := gofakeit.Price(100, 10000)
	deliveryCost := gofakeit.Price(0, 500)

	daysAgo := gofakeit.Number(1, 30)
	hoursAgo := gofakeit.Number(1, 24)
	paymentTime := time.Now().Add(-time.Duration(daysAgo)*24*time.Hour - time.Duration(hoursAgo)*time.Hour)

	return Payment{
		Transaction:  gofakeit.UUID(),
		RequestID:    gofakeit.UUID(),
		Currency:     gofakeit.CurrencyShort(),
		Provider:     gofakeit.RandomString([]string{"wbpay", "stripe", "paypal"}),
		Amount:       goodsTotal + deliveryCost,
		PaymentDT:    paymentTime.Unix(),
		Bank:         gofakeit.RandomString([]string{"alpha", "sber", "tinkoff", "vtb"}),
		DeliveryCost: deliveryCost,
		GoodsTotal:   goodsTotal,
		CustomFee:    gofakeit.Price(0, 100),
	}
}

// createItem выдаёт экземляр структуры Item
func createItem() Item {

	price := gofakeit.Price(10, 1000)
	sale := gofakeit.Float64Range(0, 50)

	return Item{
		ChrtID:      gofakeit.Number(1000000, 9999999),
		TrackNumber: "WBILMTESTTRACK",
		Price:       price,
		RID:         gofakeit.UUID(),
		Name:        gofakeit.ProductName(),
		Sale:        sale,
		Size:        gofakeit.RandomString([]string{"S", "M", "L", "XL"}),
		TotalPrice:  price * (1 - sale/100),
		NMID:        gofakeit.Number(1000000, 9999999),
		Brand:       gofakeit.Company(),
		Status:      gofakeit.Number(200, 204),
	}
}

// messageGenerate организует псевдослучайные данные для передачи брокеру
func messageGenerate(count int) [][]byte {

	start := time.Now()

	// создаем span для генерации батча сообщений
	ctx, span := tracer.Start(context.Background(), "producer.generate.batch",
		trace.WithAttributes(
			attribute.Int("batch.size", count),
			attribute.String("component", "producer"),
		))
	defer span.End()

	testMsg := make([][]byte, count)

	for i := 0; i < count; i++ {

		// создаем span для генерации одного сообщения
		_, msgSpan := tracer.Start(ctx, "producer.generate.message",
			trace.WithAttributes(
				attribute.Int("message.index", i),
			))

		delivery := createDelivery()
		payment := createPayment()
		items := []Item{createItem()}

		// иногда добавляем несколько товаров
		if rand.Intn(2) == 0 {
			items = append(items, createItem())
		}

		order := Order{
			OrderUID:          payment.Transaction,
			TrackNumber:       "WBILMTESTTRACK",
			Entry:             "WBIL",
			Delivery:          delivery,
			Payment:           payment,
			Items:             items,
			Locale:            gofakeit.RandomString([]string{"en", "ru"}),
			InternalSignature: "",
			CustomerID:        gofakeit.UUID(),
			DeliveryService:   gofakeit.RandomString([]string{"meest", "cdek", "dhl", "ups"}),
			Shardkey:          fmt.Sprintf("%d", gofakeit.Number(0, 9)),
			SMID:              gofakeit.Number(1, 99),
			DateCreated:       time.Unix(payment.PaymentDT, 0),
			OOFShard:          fmt.Sprintf("%d", gofakeit.Number(0, 5)),
		}

		orderInByte, err := json.Marshal(order)
		if err != nil {
			log.Printf("Ошибка маршалинга заказа: %v.\n", err)
			msgSpan.End()
			continue
		}

		testMsg[i] = orderInByte

		// записываем метрику: одно сообщение сгенерировано
		messagesGenerated.Inc()

		msgSpan.End()
	}

	// записываем метрику времени генерации батча
	duration := time.Since(start).Seconds()
	generateDuration.Observe(duration)

	span.SetAttributes(attribute.Int("generated.count", len(testMsg)))

	return testMsg
}
