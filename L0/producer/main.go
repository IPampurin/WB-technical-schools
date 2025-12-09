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
)

// выносим константы конфигурации по умолчанию, чтобы были на виду

const (
	topicNameConst     = "my-topic" // имя топика, коррелируется с консумером
	kafkaPortConst     = 9092       // порт, на котором сидит kafka по умолчанию
	massagesCountConst = 100        // количество сообщений, отправляемых одним врайтером, по умолчанию
	writersCountConst  = 50         // количество врайтеров для имитации отправки "со всех сторон", по умолчанию
)

// ProducerConfig описывает настройки с учётом переменных окружения
type ProducerConfig struct {
	Topic         string // имя топика (коррелируется с консумером)
	KafkaPort     int    // порт, на котором сидит kafka
	MassagesCount int    // количество сообщений, отправляемых одним врайтером
	WritersCount  int    // количество врайтеров
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
func readConfig() *ProducerConfig {

	return &ProducerConfig{
		Topic:         getEnvString("TOPIC_NAME_STR", topicNameConst),
		KafkaPort:     getEnvInt("KAFKA_PORT_NUM", kafkaPortConst),
		MassagesCount: getEnvInt("MESAGES_COUNT", massagesCountConst),
		WritersCount:  getEnvInt("WRITERS_COUNT", writersCountConst),
	}
}

// sendMessages генерирует и отправляет брокеру заданное количество сообщений
func sendMessages(ctx context.Context, w *kafka.Writer, cfg *ProducerConfig, generatedCount, countSended, failedCount *int64, wg *sync.WaitGroup) {

	defer wg.Done()

	// генерируем сообщения для отправки
	messages := messageGenerate(cfg.MassagesCount)
	if len(messages) == 0 {
		log.Println("Не сгенерировано ни одного сообщения для отправки.")
		return
	}

	atomic.AddInt64(generatedCount, int64(len(messages)))

	// отправляем сообщения с проверкой контекста
	for i, msgBody := range messages {

		select {
		default:
			msg := kafka.Message{
				Key:   []byte(fmt.Sprintf("Сообщение №%d", i+1)),
				Value: msgBody,
				Time:  time.Now(),
			}

			err := w.WriteMessages(context.Background(), msg)
			if err != nil {
				atomic.AddInt64(failedCount, 1)
			} else {
				atomic.AddInt64(countSended, 1)
			}
		case <-ctx.Done():
			return
		}
	}
}

func main() {

	// инициализируем генератор gofakeit
	gofakeit.Seed(0)

	// считываем конфигурацию
	cfg := readConfig()

	// устанавливаем соединение с брокером
	conn, err := kafka.DialLeader(context.Background(), "tcp", fmt.Sprintf("localhost:%d", cfg.KafkaPort), cfg.Topic, 0)
	if err != nil {
		log.Fatalf("ошибка создания топика кафки: %v\n", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Ошибка при закрытии продюсером соединения с  кафкой: %v", err)
		}
	}()

	log.Println("Соединение с брокером установлено.")

	// создаём ряд врайтеров
	writers := make([]*kafka.Writer, cfg.WritersCount)
	for i := 0; i < len(writers); i++ {
		// определяем продюсер
		writers[i] = &kafka.Writer{
			Addr:         kafka.TCP(fmt.Sprintf("localhost:%d", cfg.KafkaPort)), // список брокеров
			Topic:        cfg.Topic,                                             // имя топика, в который будем слать сообщения
			Async:        false,                                                 // можно установить true и получить максимальную скорость без гарантии доставки
			RequiredAcks: kafka.RequireAll,                                      // максимальный контроль доставки (подтверждение от всех реплик)
		}
		defer func() {
			if err := writers[i].Close(); err != nil {
				log.Printf("Ошибка при закрытии продюсера: %v", err)
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

	var wg sync.WaitGroup

	for i := 0; i < len(writers); i++ {
		wg.Add(1)
		go sendMessages(ctx, writers[i], cfg, &generatedCount, &countSended, &failedCount, &wg)
	}

	wg.Wait()

	log.Println("Врайтеры завершили отправку сообщений брокеру.")

	log.Printf("Сгенерировано сообщений: %d. Отправлено брокеру сообщений: %d. Ошибок при отправке: %d.\n", generatedCount, countSended, failedCount)
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

	testMsg := make([][]byte, count)

	for i := 0; i < count; i++ {
		delivery := createDelivery()
		payment := createPayment()
		items := []Item{createItem()}

		// Иногда добавляем несколько товаров
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
			log.Printf("Ошибка маршалинга заказа: %v", err)
			continue
		}

		testMsg[i] = orderInByte
	}

	return testMsg
}
