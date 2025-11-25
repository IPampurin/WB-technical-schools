package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/segmentio/kafka-go"
)

var (
	topic          = "my-topic-L0"          // имя топика, в который пишем сообщения
	countMessage   = 10                     // количество тестовых джасончиков
	maxRetries     = 3                      // количество повторных попыток связи
	retryDelayBase = 100 * time.Millisecond // базовая задержка для попыток связи
)

// sendWithRetry отправляет сообщения с повторами
func sendWithRetry(ctx context.Context, w *kafka.Writer, msg kafka.Message, msgNumber int) bool {

	for attempt := 1; attempt <= maxRetries; attempt++ {

		err := w.WriteMessages(ctx, msg)
		if err == nil {
			return true // успешная отправка
		}

		// проверяем отмену контекста
		if errors.Is(err, context.Canceled) {
			log.Printf("Отправка прервана на сообщении %d (попытка %d)", msgNumber, attempt)
			return false
		}

		log.Printf("Ошибка отправки сообщения №%d (попытка %d/%d): %v", msgNumber, attempt, maxRetries, err)

		// проверяем номер попытки и делаем паузу перед следующей попыткой
		if attempt < maxRetries {

			// высчитываем увеличивающуюся паузу (200ms, 600ms, 1200ms)
			delay := retryDelayBase * time.Duration(attempt*attempt+attempt)
			log.Printf("Повторная отправка сообщения №%d через %v...", msgNumber, delay)

			select {
			case <-time.After(delay):
				// продолжаем следующую попытку
			case <-ctx.Done():
				log.Printf("Отправка прервана во время ожидания ретрая для сообщения %d", msgNumber)
				return false
			}
		}
	}

	// все попытки исчерпаны
	log.Printf("Сообщение №%d не отправлено после %d попыток", msgNumber, maxRetries)

	return false
}

func main() {

	// инициализируем генератор gofakeit
	gofakeit.Seed(0)

	// устанавливаем соединение с брокером
	conn, err := kafka.DialLeader(context.Background(), "tcp", "localhost:9092", topic, 0)
	if err != nil {
		log.Fatalf("ошибка создания топика кафки: %v\n", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Ошибка при закрытии продюсером соединения с  кафкой: %v", err)
		}
	}()

	log.Println("Соединение с брокером установлено.")

	// определяем продюсер
	w := &kafka.Writer{
		Addr:  kafka.TCP("localhost:9092"), // список брокеров
		Topic: topic,                       // имя топика, в который будем слать сообщения
		// RequiredAcks: kafka.WaitForLocal	// по умолчанию лидер в кафке подтверждает получение сообщения
	}
	defer func() {
		if err := w.Close(); err != nil {
			log.Printf("Ошибка при закрытии продюсера: %v", err)
		}
	}()

	log.Println("Начинаем генерировать тестовые данные.")

	// собираем и отправляем тестовые сообщения, если они есть
	messages := messageGenerate(countMessage)
	if len(messages) == 0 {
		log.Println("Не сгенерировано ни одного сообщения для отправки")
		return
	}

	log.Printf("Сгенерировано %d сообщений. Начинаем отправку...", len(messages))

	// организуем контекст для корректного завершения продюсера,
	// например, если мы его в контейнере докера гасим
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// обработка сигналов
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// фоном слушаем сигналы отмены и отменяем контекст
	go func() {
		<-sigChan
		log.Println("Получен сигнал остановки, завершаем отправку...")
		cancel()
	}()

	log.Println("Начинаем отправку тестовых данных.")

	sentCount := 0   // количество отправленных сообщений
	failedCount := 0 // количество ошибок при отправке сообщений

	// отправляем сообщения с проверкой контекста
	for i, msgBody := range messages {

		// проверяем контекст перед началом обработки каждого сообщения
		if ctx.Err() != nil {
			log.Printf("Отправка прервана. Успешно отправлено: %d/%d, не отправлено: %d",
				sentCount, len(messages), len(messages)-sentCount)
			return
		}

		msg := kafka.Message{
			Key:   []byte(fmt.Sprintf("Сообщение №%d", i+1)),
			Value: msgBody,
			Time:  time.Now(),
		}

		// отправляем сообщение с ретраями
		success := sendWithRetry(ctx, w, msg, i+1)
		if success {
			fmt.Printf("Успешно отправлено сообщение %d: %s\n", i+1, string(msgBody))
			sentCount++
		} else {
			failedCount++
		}
	}

	// финальная запись в логи
	if failedCount > 0 {
		log.Printf("Отправка завершена. Успешно: %d/%d, окончательных ошибок: %d",
			sentCount, len(messages), failedCount)
	} else {
		log.Printf("Продюсер успешно отправил все %d сообщений.", sentCount)
	}
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
			DateCreated:       time.Unix(payment.PaymentDT, 0), // Синхронизируем с PaymentDT
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
