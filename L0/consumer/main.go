package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	topic   = "my-topic-L0"
	groupID = "my-groupID"
)

func main() {

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{"localhost:9092"},
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 0,
		MaxWait:  10 * time.Second,
	})
	defer r.Close()

	httpClient := &http.Client{}
	apiURL := "http://localhost:8081/order" // URL сервера

	fmt.Printf("Консьюмер подписан на топик '%s' в группе '%s'\n\n", topic, groupID)

	fmt.Println("начинаем вычитывать !!!")

	ctx := context.Background()
	for {
		m, err := r.FetchMessage(ctx)
		if err != nil {
			log.Printf("ошибка чтения из кафки: %v\n", err)
			continue
		}

		// Отправка сообщения через HTTP POST
		resp, err := httpClient.Post(
			apiURL,
			"application/json",
			bytes.NewReader(m.Value),
		)

		fmt.Printf("%s: %s\n\n", string(m.Key), string(m.Value))

		if err == nil && resp.StatusCode == http.StatusOK {
			if err := r.CommitMessages(ctx, m); err != nil {
				log.Fatal("failed to commit messages:", err)
			} else {
				log.Printf("Сообщение успешно обработано: %s", m.Key)
			}
		} else {
			log.Printf("Ошибка обработки: %v", err)
		}

		if resp != nil {
			resp.Body.Close()
		}
	}
}

/*

// consumer/main.go
func main() {
    // ... (инициализация ридера Kafka)

    httpClient := &http.Client{Timeout: 5 * time.Second}
    apiURL := "http://localhost:8081/order" // URL сервера

    for {
        m, err := r.FetchMessage(ctx)
        if err != nil {
            log.Printf("Ошибка чтения: %v", err)
            continue
        }

        // Отправка сообщения через HTTP POST
        resp, err := httpClient.Post(
            apiURL,
            "application/json",
            bytes.NewReader(m.Value),
        )

        if err == nil && resp.StatusCode == http.StatusOK {
            // Коммит только при успешной обработке
            r.CommitMessages(ctx, m)
            log.Printf("Сообщение успешно обработано: %s", m.Key)
        } else {
            log.Printf("Ошибка обработки: %v", err)
        }

        if resp != nil {
            resp.Body.Close()
        }
    }
}


*/
