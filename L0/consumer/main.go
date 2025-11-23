package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
)

var (
	topic   = "my-topic-L0" // имя топика коррелируется с продюсером
	groupID = "my-groupID"  // произвольное в нашем случае имя группы
)

// consumer это основной код консумера
func consumer(ctx context.Context) error {

	// организуем клиента для отправки вычитанных из кафки сообщений на api сервиса
	// по умолчанию порт хоста 8081 (доступ в браузере на localhost:8081)
	port, ok := os.LookupEnv("L0_PORT")
	if !ok {
		port = "8081"
	}

	apiURL := fmt.Sprintf("http://localhost:%s/order", port)
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// ридер из кафки
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{"localhost:9092"},
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 0,
		MaxWait:  10 * time.Second,
	})
	defer func() {
		if err := r.Close(); err != nil {
			log.Printf("Ошибка при закрытии ридера Kafka: %v", err)
		}
	}()

	log.Printf("Консуюмер подписан на топик '%s' в группе '%s'\n", topic, groupID)
	log.Println("Начинаем вычитывать !!!")

	// смотрим что прислали
	for ctx.Err() == nil {

		m, err := r.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Println("Получен сигнал завершения работы консумера.")
				return nil
			}
			log.Printf("ошибка чтения из кафки: %v\n", err)
			continue
		}

		// отправляем сообщения через HTTP POST на api приложения
		resp, err := httpClient.Post(apiURL, "application/json", bytes.NewReader(m.Value))
		if err != nil {
			log.Printf("ошибка сети: %v", err)
			// продолжаем работу несмотря на сетевые ошибки
			continue
		}

		if 200 <= resp.StatusCode && resp.StatusCode < 300 || resp.StatusCode == http.StatusConflict { // статусы 2XX = ок и 409 = дубль
			if err := r.CommitMessages(ctx, m); err != nil {
				if errors.Is(err, context.Canceled) {
					log.Println("Получен сигнал завершения работы консумера при коммите.")
					return nil
				}
				log.Printf("ошибка при коммите сообщения %s\n: %v", m.Key, err)
				return fmt.Errorf("ошибка коммита сообщения: %w", err)
			} else {
				log.Printf("Сообщение успешно обработано и закоммичено: %s\n", m.Key)
			}
		} else {
			log.Printf("Сообщение обработано, не закоммиченно, статус ответа: %v\n", resp.StatusCode) // статусы Bad
			if resp.StatusCode >= 500 {
				return fmt.Errorf("серверная ошибка API: статус %d", resp.StatusCode)
			}
		}

		resp.Body.Close() // закрываем тело запроса

	}

	return nil
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeLimitConsumer)*time.Second)
	defer cancel()

	// обработка сигналов ОС для graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// запускаем консумер в горутине
	consumerDone := make(chan error, 1)
	go func() {
		consumerDone <- consumer(ctx)
	}()

	// ожидаем либо сигнал ОС, либо завершение консумера по timeLimitConsumer
	select {
	case sig := <-sigChan:
		log.Printf("Получен сигнал: %v. Завершаем работу...", sig)
		cancel() // отменяем контекст

		// даем время на graceful shutdown (30 секунд)
		select {
		case <-consumerDone:
			log.Println("Консумер корректно завершил работу")
		case <-time.After(30 * time.Second):
			log.Println("Таймаут graceful shutdown, принудительное завершение")
		}

	case err := <-consumerDone:
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			log.Printf("Консумер завершился с ошибкой: %v", err)
		} else if errors.Is(err, context.DeadlineExceeded) {
			log.Println("Консумер завершил работу по таймауту")
		} else {
			log.Println("Консумер завершил работу")
		}
	}

	log.Println("Консумер завершил работу.")
}
