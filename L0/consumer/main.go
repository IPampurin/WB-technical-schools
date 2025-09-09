package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"
)

var (
	topic   = "my-topic-L0" // имя топика коррелируется с продюсером
	groupID = "my-groupID"  // произвольное в нашем случае имя группы
)

// consumer это основной код консумера
func consumer(ctx context.Context) {

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

	// смотрим что прислали
	for ctx.Err() == nil {

		m, err := r.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Println("Получен сигнал завершения работы консумера.")
				break
			}
			log.Printf("ошибка чтения из кафки: %v\n", err)
			continue
		}

		// отправляем сообщения через HTTP POST на api приложения
		resp, err := httpClient.Post(apiURL, "application/json", bytes.NewReader(m.Value))
		if err != nil {
			log.Printf("ошибка сети: %v", err)
			continue
		}

		if 200 <= resp.StatusCode && resp.StatusCode < 300 || resp.StatusCode == http.StatusConflict { // статусы 2XX = ок и 409 = дубль
			if err := r.CommitMessages(ctx, m); err != nil {
				log.Printf("ошибка при коммите сообщения %s\n: %v", m.Key, err)
			} else {
				log.Printf("Сообщение успешно обработано и закоммичено: %s\n", m.Key)
			}
		} else {
			log.Printf("Сообщение обработано, не закоммиченно, статус ответа: %v\n", resp.StatusCode) // статусы Bad
		}

		resp.Body.Close() // закрываем тело запроса

	}
}

func main() {

	// определяем время работы консумера
	limitStr, ok := os.LookupEnv("TIME_LIMIT_CONSUMER_L0")
	if !ok {
		limitStr = "10800"
	}

	timeLimitConsumer, err := strconv.Atoi(limitStr)
	if err != nil {
		log.Printf("проверьте .env файл, ошибка назначения времени работы консумера: %v\n", err)
		log.Println("Время работы консумера принимается 3 часа.")
		timeLimitConsumer = 10800
	}

	// заведём контекст для отмены работы консумера
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// через определённое время завершаем работу консумера
	go func() {
		<-time.After(time.Duration(timeLimitConsumer) * time.Second)
		cancel()
	}()

	// запускаем основной код консумера
	consumer(ctx)

	log.Println("Консумер завершил работу.")
}
