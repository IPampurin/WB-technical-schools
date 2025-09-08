package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/segmentio/kafka-go"
)

var (
	topic   = "my-topic-L0" // имя топика коррелируется с продюсером
	groupID = "my-groupID"  // произвольное в нашем случае имя группы
)

func main() {

	// организуем клиента для отправки вычитанных из кафки сообщений на api сервиса
	// по умолчанию порт хоста 8081 (доступ в браузере на localhost:8081)
	port, ok := os.LookupEnv("L0_PORT")
	if !ok {
		port = "8081"
	}

	apiURL := fmt.Sprintf("http://localhost:%s/order", port)
	httpClient := &http.Client{}

	// ридер из кафки
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{"localhost:9092"},
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 0,
		MaxWait:  10 * time.Second,
	})
	defer r.Close()

	log.Printf("Консуюмер подписан на топик '%s' в группе '%s'\n", topic, groupID)
	log.Println("Начинаем вычитывать !!!")

	ctx := context.Background()
	for {
		m, err := r.FetchMessage(ctx)
		if err != nil {
			log.Printf("ошибка чтения из кафки: %v\n", err)
			continue
		}

		// отправляем сообщения через HTTP POST на api приложения
		resp, err := httpClient.Post(apiURL, "application/json", bytes.NewReader(m.Value))
		if err != nil {
			log.Printf("Сетевая ошибка: %v", err)
		} else {
			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusConflict { // статусы 200 = ок и 409 = дубль
				if err := r.CommitMessages(ctx, m); err != nil {
					log.Fatal("failed to commit messages:", err)
				} else {
					log.Printf("Сообщение успешно обработано: %s", m.Key)
				}
			} else {
				log.Printf("Сообщение обработано, статус ответа: %v", resp.StatusCode) // статус 201 = создана запись
			}
		}

		if resp != nil {
			resp.Body.Close()
		}
	}
}
