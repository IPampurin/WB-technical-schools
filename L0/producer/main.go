package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"
	"unicode"

	"github.com/segmentio/kafka-go"
)

const (
	topic        = "my-topic-L0" // имя топика, в который пишем сообщения
	countMessage = 5             // количество тестовых джасончиков
)

func main() {

	// устанавливаем соединение с брокером
	conn, err := kafka.DialLeader(context.Background(), "tcp", "localhost:9092", topic, 0)
	if err != nil {
		log.Fatalf("ошибка создания топика кафки: %v\n", err)
	}
	defer conn.Close()

	// определяем продюсер
	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   topic,
	})
	defer w.Close()

	// собираем и отправляем тестовые сообщения
	messages := messageGenerate()

	for i, msgBody := range messages {
		msg := kafka.Message{
			Key:   []byte(fmt.Sprintf("Сообщение №%d", i+1)),
			Value: msgBody,
			Time:  time.Now(),
		}

		// отправляем сообщения с дефолтным условием подтверждения получения от лидера
		err := w.WriteMessages(context.Background(), msg)
		if err != nil {
			log.Printf("ошибка отправления сообщения в кафку '%s': %v\n", msgBody, err)
		} else {
			fmt.Printf("Отправленное в кафку сообщение: %s\n", string(msgBody))
		}
	}

	fmt.Println("Producer finished.")
}

// Заказ
type Order struct {
	OrderUID          string    `json:"order_uid" gorm:"unique_index;not null"`
	TrackNumber       string    `json:"track_number" gorm:"index"`
	Entry             string    `json:"entry"`
	Delivery          *Delivery `json:"delivery" gorm:"foreignKey:OrderID"`
	Payment           *Payment  `json:"payment" gorm:"foreignKey:OrderID"`
	Items             []*Item   `json:"items" gorm:"foreignKey:OrderID"`
	Locale            string    `json:"locale"`
	InternalSignature string    `json:"internal_signature"`
	CustomerID        string    `json:"customer_id"`
	DeliveryService   string    `json:"delivery_service"`
	Shardkey          string    `json:"shardkey"`
	SMID              int       `json:"sm_id" gorm:"type:smallint"`
	DateCreated       time.Time `json:"date_created" gorm:"type:timestamp;default:CURRENT_TIMESTAMP"`
	OOFShard          string    `json:"oof_shard"`
}

// Доставка
type Delivery struct {
	OrderID uint   `json:"-"`
	Name    string `json:"name" gorm:"size:100"`
	Phone   string `json:"phone" gorm:"size:20"`
	Zip     string `json:"zip" gorm:"size:20"`
	City    string `json:"city" gorm:"size:100"`
	Address string `json:"address" gorm:"size:255"`
	Region  string `json:"region" gorm:"size:100"`
	Email   string `json:"email" gorm:"unique_index;type:varchar(100)"`
}

// Оплата
type Payment struct {
	OrderID      uint    `json:"-"`
	Transaction  string  `json:"transaction" gorm:"unique_index"`
	RequestID    string  `json:"request_id"`
	Currency     string  `json:"currency" gorm:"size:3"`
	Provider     string  `json:"provider" gorm:"size:50"`
	Amount       float64 `json:"amount" gorm:"type:decimal(10,2)"`
	PaymentDT    int64   `json:"payment_dt" gorm:"type:bigint"`
	Bank         string  `json:"bank" gorm:"size:50"`
	DeliveryCost float64 `json:"delivery_cost" gorm:"type:decimal(10,2)"`
	GoodsTotal   float64 `json:"goods_total" gorm:"type:decimal(10,2)"`
	CustomFee    float64 `json:"custom_fee" gorm:"type:decimal(10,2)"`
}

// Позиция заказа
type Item struct {
	OrderID     uint    `json:"-"`
	ChrtID      int     `json:"chrt_id" gorm:"type:integer"`
	TrackNumber string  `json:"track_number"`
	Price       float64 `json:"price" gorm:"type:decimal(10,2)"`
	RID         string  `json:"rid" gorm:"size:100"`
	Name        string  `json:"name" gorm:"size:255"`
	Sale        float64 `json:"sale" gorm:"type:decimal(5,2)"`
	Size        string  `json:"size" gorm:"size:20"`
	TotalPrice  float64 `json:"total_price" gorm:"type:decimal(10,2)"`
	NMID        int     `json:"nm_id" gorm:"type:integer"`
	Brand       string  `json:"brand" gorm:"size:100"`
	Status      int     `json:"status" gorm:"type:smallint"`
}

// createDelivery выдаёт указатель на экземляр структуры Delivery
func createDelivery() *Delivery {

	delivery := &Delivery{
		Name:    generateWord() + " " + generateWord(),                                         // string
		Phone:   generatePhone(),                                                               // string
		Zip:     strconv.Itoa(int(generateNumber(2639809))),                                    // string
		City:    generateWord() + " " + generateWord(),                                         // string
		Address: generateWord() + " " + generateWord() + strconv.Itoa(int(generateNumber(15))), // string
		Region:  generateWord(),                                                                // string
		Email:   generateWord() + "@gmail.com",                                                 // string
	}

	return delivery
}

// createPayment выдаёт указатель на экземляр структуры Payment
func createPayment() *Payment {

	goodsTotal := generateNumber(317)    // придумываем цену
	deliveryCost := generateNumber(1500) // придумываем цену доставки

	payment := &Payment{
		Transaction:  generateRid(),                     // string
		RequestID:    "",                                // string
		Currency:     "USD",                             // string
		Provider:     "wbpay",                           // string
		Amount:       deliveryCost + goodsTotal,         // float64
		PaymentDT:    int64(generateNumber(1637907727)), // int64
		Bank:         "alpha",                           // string
		DeliveryCost: deliveryCost,                      // float64
		GoodsTotal:   goodsTotal,                        // float64
		CustomFee:    0,                                 // float64
	}

	return payment
}

// createItem выдаёт указатель на экземляр структуры Item
func createItem() *Item {

	price := generateNumber(10000)  // придумываем цену
	sale := generateNumber(100)     // придумываем скидку
	size := int(generateNumber(60)) // придумываем размер

	item := &Item{
		ChrtID:      int(generateNumber(9934930)),          // int
		TrackNumber: "WBILMTESTTRACK",                      // const
		Price:       price,                                 // float64
		RID:         generateRid(),                         // string
		Name:        generateWord(),                        // string
		Sale:        sale,                                  // float64
		Size:        strconv.Itoa(size),                    // string
		TotalPrice:  price * (1 - (sale / 100)),            // float64
		NMID:        int(generateNumber(2389212)),          // int
		Brand:       generateWord() + " " + generateWord(), // string
		Status:      202,                                   // int
	}

	return item
}

// generateNumber генерирует случайное число в диапазоне [0, max)
func generateNumber(max int) float64 {

	return float64(rand.Float64() * float64(max))
}

// generateWord генерирует слово на латинице с заглавной буквы длиной от 4 до 10 букв
func generateWord() string {

	letters := "abcdefghijklmnopqrstuvwxyz" // набор возможных букв для генерации слова

	length := rand.Intn(7) + 4 // генерируем случайную длину слова от 4 до 10 букв

	word := make([]rune, 0, 10)

	// заполняем слово случайными буквами
	for i := 0; i < length; i++ {
		randomIndex := rand.Intn(len(letters))
		randomLetter := rune(letters[randomIndex])
		if i == 0 {
			randomLetter = unicode.ToUpper(randomLetter) // если буква не первая, меняем регистр
		}
		word = append(word, randomLetter)
	}

	return string(word)
}

// generateRid генерирует строку в формате "случайная_часть + слово"
func generateRid() string {

	letters := "abcdefghijklmnopqrstuvwxyz0123456789" // набор возможных символов для случайной части

	randomPartLength := 19 // фиксированная длина случайной части

	fixedWord := "test" // суфикс

	randomPart := make([]rune, 0, randomPartLength)

	// генерируем случайную часть
	for i := 0; i < randomPartLength; i++ {
		randomIndex := rand.Intn(len(letters))
		randomLetter := rune(letters[randomIndex])
		randomPart = append(randomPart, randomLetter)
	}

	return string(randomPart) + fixedWord
}

// generatePhone генерирует номер телефона в формате "+ХХХХХХХХХХ"
func generatePhone() string {

	number := make([]byte, 10, 10)

	for i := 0; i < 10; i++ {
		digit := rand.Intn(10)        // генерируем цифру от 0 до 9
		number[i] = byte(digit) + '0' // преобразуем цифру в байт
	}

	return "+" + string(number)
}

// messageGenerate организует псевдослучайные данные для передачи брокеру
func messageGenerate() [][]byte {

	testMsg := make([][]byte, countMessage, countMessage)

	var order *Order

	for i := 0; i < len(testMsg); i++ {

		delivery := createDelivery()
		payment := createPayment()
		item := createItem()

		order = &Order{
			OrderUID:          payment.Transaction,        // string
			TrackNumber:       "WBILMTESTTRACK",           // string
			Entry:             "WBIL",                     // string
			Delivery:          delivery,                   // Delivery
			Payment:           payment,                    // Payment
			Items:             []*Item{item},              // []Item. Возьмём один единственный товар, чтобы не заморачиваться
			Locale:            "en",                       // string
			InternalSignature: "",                         // string
			CustomerID:        "test",                     // string
			DeliveryService:   "meest",                    // string
			Shardkey:          strconv.Itoa(rand.Intn(9)), // string
			SMID:              rand.Intn(99),              // int
			DateCreated:       time.Now().UTC(),           // time.Time
			OOFShard:          strconv.Itoa(rand.Intn(2)), // string
		}

		orderInByte, err := json.Marshal(order) // превращаем экземпляр order в []byte
		if err != nil {
			continue // тут ошибка нам не интересна - просто пропустим досадную неожиданность
		}

		testMsg[i] = orderInByte
	}

	return testMsg
}
