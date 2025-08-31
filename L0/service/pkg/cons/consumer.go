package consumer

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/segmentio/kafka-go"
)

func Consumer() {

	topic, ok := os.LookupEnv("TOPIC_L0")
	if !ok {
		topic = "my-topic"
	}
	groupID, ok := os.LookupEnv("GroupID_L0")
	if !ok {
		groupID = "my-groupID"
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{"localhost:9092"},
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 0,
		MaxWait:  10 * time.Second,
	})
	defer r.Close()

	fmt.Printf("Консьюмер подписан на топик '%s' в группе '%s'\n\n", topic, groupID)

	fmt.Println("начинаем вычитывать !!!")

	ctx := context.Background()
	for {
		m, err := r.FetchMessage(ctx)
		if err != nil {
			log.Fatalf("ошибка чтения из кафки: %v\n", err)
		}
		fmt.Printf("%s: %s\n\n", string(m.Key), string(m.Value))
		if err := r.CommitMessages(ctx, m); err != nil {
			log.Fatal("failed to commit messages:", err)
		}
	}
}
