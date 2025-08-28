package cons

import (
	"bytes"
	"log"
	"net/http"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

// Конфигурация подключения
const (
	kafkaBrokers = "localhost:9092"
	kafkaTopic   = "your-topic"
	kafkaGroupID = "your-group-id"
	apiURL       = "http://localhost:8080/api/orders"
)

func main() {
	// Создаем клиента для работы с API
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	// Создаем консьюмера
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": kafkaBrokers,
		"group.id":          kafkaGroupID,
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer consumer.Close()

	// Подписка на топик
	consumer.SubscribeTopics([]string{kafkaTopic}, nil)

	for {
		msg, err := consumer.ReadMessage(-1)
		if err == nil {
			go processMessage(client, msg.Value)
		} else {
			log.Printf("Error reading message: %v\n", err)
		}
	}
}

func processMessage(client *http.Client, message []byte) {
	// Формируем HTTP-запрос
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(message))
	if err != nil {
		log.Printf("Error creating request: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	// Отправляем запрос
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		log.Printf("Failed to create order: %d\n", resp.StatusCode)
	}
}
