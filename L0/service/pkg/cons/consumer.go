package cons

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	topic   = "my-topic-L0"
	groupID = "my-groupID"
)

func Consumer() {

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

// Заказ
type Order struct {
	OrderUID          string    `json:"order_uid" gorm:"unique_index;not null"`
	TrackNumber       string    `json:"track_number" gorm:"index"`
	Entry             string    `json:"entry"`
	Delivery          Delivery  `json:"delivery" gorm:"foreignKey:OrderID"`
	Payment           Payment   `json:"payment" gorm:"foreignKey:OrderID"`
	Items             []Item    `json:"items" gorm:"foreignKey:OrderID"`
	Locale            string    `json:"locale"`
	InternalSignature string    `json:"internal_signature"`
	CustomerID        string    `json:"customer_id"`
	DeliveryService   string    `json:"delivery_service"`
	Shardkey          string    `json:"shardkey"`
	SMID              int       `json:"sm_id" gorm:"type:smallint"`
	DateCreated       time.Time `json:"date_created" gorm:"type:timestamp;default:CURRENT_TIMESTAMP"`
	OOFShard          string    `json:"oof_shard"`
}
