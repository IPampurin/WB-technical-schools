package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

const topic = "my-topic-L0"

func main() {

	conn, err := kafka.DialLeader(context.Background(), "tcp", "localhost:9092", topic, 0)
	if err != nil {
		log.Fatalf("ошибка создания топика кафки: %v\n", err)
	}
	defer conn.Close()

	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   topic,
	})
	defer w.Close()

	msgs := []string{
		"test 6",
		"test 7",
		"test 8",
		"test 9",
		"test 10",
	}

	for i, msgBody := range msgs {
		msg := kafka.Message{
			Key:   []byte(fmt.Sprintf("Сообщение №%d", i+1)),
			Value: []byte(msgBody),
			Time:  time.Now(),
		}

		err := w.WriteMessages(context.Background(), msg)
		if err != nil {
			log.Printf("ошибка отправления сообщения в кафку '%s': %v\n", msgBody, err)
		} else {
			fmt.Printf("Отправленное в кафку сообщение: %s\n", msgBody)
		}
		time.Sleep(10 * time.Millisecond)
	}

	fmt.Println("Producer finished.")
}
