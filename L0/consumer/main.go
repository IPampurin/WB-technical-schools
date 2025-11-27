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
	topic          = "my-topic-L0"          // имя топика коррелируется с продюсером
	groupID        = "my-groupID"           // произвольное в нашем случае имя группы
	maxRetries     = 3                      // количество повторных попыток связи
	retryDelayBase = 100 * time.Millisecond // базовая задержка для попыток связи
	clientTimeout  = 30 * time.Second       // таймаут для HTTP клиента
)

// sendWithRetry передает сообщение в API с повторами
func sendWithRetry(ctx context.Context, apiUrl string, data []byte) (*http.Response, error) {

	// организуем клиента для отправки вычитанных из кафки сообщений на api сервиса
	httpClient := &http.Client{
		Timeout: clientTimeout,
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {

		// шлём запрос на сервер и получаем ответ
		resp, err := httpClient.Post(apiUrl, "application/json", bytes.NewReader(data))
		if err == nil {
			return resp, nil // успешная отправка (закрываем resp.Body далее при обработке)
		}

		log.Printf("Ошибка сети (попытка %d/%d): %v", attempt, maxRetries, err)

		// проверяем номер попытки и делаем паузу перед следующей попыткой
		if attempt < maxRetries {

			// высчитываем увеличивающуюся паузу (200ms, 600ms, 1200ms)
			delay := retryDelayBase * time.Duration(attempt*attempt+attempt)
			log.Printf("Попытка отправки %v", attempt)

			select {
			case <-time.After(delay):
				// продолжаем следующую попытку
			case <-ctx.Done():
				log.Println("Прерываем повторы отправки сообщений при отмене контекста.")
				return nil, ctx.Err()
			}
		}
	}

	return nil, fmt.Errorf("не удалось отправить запрос после %d попыток", maxRetries)
}

// handleAPIResponse обрабатывает ответ API и решает коммитить ли сообщение
func handleAPIResponse(ctx context.Context, r *kafka.Reader, m kafka.Message, resp *http.Response) (bool, error) {

	defer resp.Body.Close()

	switch {

	// успешные ответы - коммитим
	case resp.StatusCode >= 200 && resp.StatusCode < 300: // 200-299
		fallthrough
	case resp.StatusCode == http.StatusConflict: // 409 - дубль
		if err := r.CommitMessages(ctx, m); err != nil {
			if errors.Is(err, context.Canceled) {
				return false, fmt.Errorf("Прерываем коммит сообщения при отмене контекста: %w", err)
			}
			return false, fmt.Errorf("ошибка коммита: %w", err)
		}
		log.Printf("Сообщение обработано и закоммичено: %s (статус: %d)", m.Key, resp.StatusCode)
		return true, nil

		// редиректы - не коммитим, пробуем снова
	case resp.StatusCode >= 300 && resp.StatusCode < 400: // 300-399
		log.Printf("Получен редирект: %d для %s", resp.StatusCode, m.Key)
		return false, nil

	// клиентские ошибки - не коммитим, пропускаем сообщение
	case resp.StatusCode >= 400 && resp.StatusCode < 500: // 400-499 (кроме 409)
		log.Printf("клиентская ошибка: %d для %s", resp.StatusCode, m.Key)
		return false, nil

	// серверные ошибки - критические
	case resp.StatusCode >= 500 && resp.StatusCode < 600: // 500-599
		return false, fmt.Errorf("серверная ошибка API: статус %d", resp.StatusCode)

	// неведомый статус
	default:
		return false, fmt.Errorf("неизвестный статус API: %d", resp.StatusCode)
	}
}

// consumer это основной код консумера
func consumer(ctx context.Context, apiUrl string) error {

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
			log.Printf("ошибка при закрытии ридера Kafka: %v", err)
		}
	}()

	log.Printf("Консумер подписан на топик '%s' в группе '%s'\n", topic, groupID)
	log.Println("Начинаем вычитывать !!!")

	processedCount := 0 // количество успешно обработанных и закоммиченных сообщений
	failedCount := 0    // количество сообщений с ошибками (кроме критических)

	// смотрим что прислали
	for {

		// проверяем контекст перед началом обработки каждого сообщения
		if ctx.Err() != nil {
			log.Println("Контекст отменен перед чтением сообщения из kafka.")
			break
		}

		// читаем сообщение
		m, err := r.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Println("Получен сигнал завершения работы консумера при чтении из kafka.")
				break
			}
			// логируем ошибку чтения (временные проблемы с kafka), но продолжаем работу
			log.Printf("ошибка чтения из кафки: %v\n", err)
			failedCount++
			continue
		}

		// отправляем сообщения через HTTP POST на api приложения
		resp, err := sendWithRetry(ctx, apiUrl, m.Value)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Println("Получена отмена контекста при отправке в api.")
				resp.Body.Close()
				break
			}
			// логируем ошибку отправки, но продолжаем работу
			// это могут быть временные сетевые проблемы
			log.Printf("ошибка отправки в api: %v", err)
			failedCount++
			continue
		}

		// обрабатываем ответ от API и решаем коммитить ли сообщение в kafka
		success, err := handleAPIResponse(ctx, r, m, resp)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Println("получена отмена контекста при обработке ответа api")
				break
			}
			// серверные ошибки API (5xx) считаем критическими
			// возвращаем ошибку чтобы завершить работу консумера
			log.Printf("критическая ошибка при обработке ответа api: %v", err)
			return fmt.Errorf("критическая ошибка консумера: %w", err)
		}

		// обновляем статистику в зависимости от результата обработки
		if success {
			processedCount++
		} else {
			failedCount++
		}
	}

	// логируем итоговую статистику обработки сообщений
	log.Printf("обработка завершена. успешно: %d, ошибок: %d", processedCount, failedCount)

	// проверяем причину завершения работы
	if errors.Is(ctx.Err(), context.Canceled) {
		log.Println("консумер завершил работу по отмене контекста")
		return nil
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		log.Println("консумер завершил работу по таймауту")
		return nil
	}
	/* Дурь
	// если контекст завершился по другой причине, возвращаем ошибку
	if ctx.Err() != nil {
		return fmt.Errorf("непредвиденное завершение консумера: %w", ctx.Err())
	}
	*/
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

	// по умолчанию порт хоста для сервиса 8081 (доступ в браузере на localhost:8081)
	port, ok := os.LookupEnv("L0_PORT")
	if !ok {
		port = "8081"
	}
	// определяем url для HTTP клиента
	apiUrl := fmt.Sprintf("http://localhost:%s/order", port)

	// заведём контекст для отмены работы консумера
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeLimitConsumer)*time.Second)
	defer cancel()

	// обработка сигналов ОС для graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// ждём сигнал отмены в фоне
	go func() {
		<-sigChan
		log.Println("Получен сигнал остановки, завершаем работу...")
		cancel()
	}()

	// запускаем основной код консумера
	err = consumer(ctx, apiUrl)
	if err != nil {
		log.Printf("консумер завершился с критической ошибкой: %v", err)
		os.Exit(1)
	}

	log.Println("Консумер корректно завершил работу.")
}
