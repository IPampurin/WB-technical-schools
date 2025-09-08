package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsumerRead(t *testing.T) {

	// задаём начальные условия
	testTopic := "test-topic-L0"
	groupID := "test-group"
	brokerAddr := "localhost:9092"

	// подключаемся к брокеру
	conn, err := kafka.DialLeader(context.Background(), "tcp", brokerAddr, testTopic, 0)
	require.NoError(t, err, "Не удалось подключиться к брокеру.")
	defer conn.Close()

	// готовим продюсер
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{brokerAddr},
		Topic:   testTopic,
	})
	defer writer.Close()

	// генерируем тестовые данные и сразу их отправляем
	expectedCount := 5 // количество тестовых сообщений
	for i := 0; i < expectedCount; i++ {
		msg := []byte(fmt.Sprintf("Тестовое сообщение %d", i+1))
		err := writer.WriteMessages(context.Background(), kafka.Message{
			Key:   []byte(fmt.Sprintf("key-%d", i)),
			Value: msg,
		})
		require.NoError(t, err, "Ошибка при отправке сообщения %d.", i+1)
	}

	// далее проверяем доставку - вычитываем сообщения

	// организуем консумер
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{brokerAddr},
		Topic:    testTopic,
		GroupID:  groupID,
		MinBytes: 0,
		MaxWait:  10 * time.Second,
	})
	defer r.Close()

	// вычитываем и считаем сообщения
	counter := 0
	for {
		_, err := r.FetchMessage(context.Background())
		if err != nil {
			require.NoError(t, err, "Ошибка при чтении сообщения.")
		}
		counter++
		if counter == expectedCount {
			break // прерываем цикл после получения всех сообщений
		}
	}

	assert.Equal(t, expectedCount, counter, "Количество прочитанных сообщений не совпадает.")

	// убираем тестовые данные
	t.Cleanup(func() {
		// создаём подключение при очистке
		conn, err := kafka.Dial("tcp", brokerAddr)
		if err != nil {
			t.Logf("Ошибка подключения при очистке: %v.", err)
			return
		}
		defer conn.Close()

		// удаляем топик
		if err := conn.DeleteTopics(testTopic); err != nil {
			t.Logf("Ошибка удаления топика: %v.", err)
		}
	})

}

func TestConsumerAPIIntegration(t *testing.T) {

	// задаём начальные условия и тестовые данные
	testTopic := "test-api-topic"
	testGroupID := "test-api-group"
	brokerAddr := "localhost:9092"
	testMessages := []string{
		`{"id":1,"data":"test1"}`,
		`{"id":2,"data":"test2"}`,
		`{"id":3,"data":"test3"}`,
	}

	// создаём подключение
	conn, err := kafka.DialLeader(context.Background(), "tcp", brokerAddr, testTopic, 0)
	require.NoError(t, err, "Не удалось создать топик")
	defer conn.Close()

	// сохраняем оригинальные значения топика и группы
	originTopic := topic
	originGroupID := groupID
	originPort := os.Getenv("L0_PORT")

	// настраиваем Cleanup
	t.Cleanup(func() {
		topic = originTopic
		groupID = originGroupID
		os.Setenv("L0_PORT", originPort)

		// удаляем тестовый топик
		conn, err := kafka.Dial("tcp", brokerAddr)
		if err == nil {
			conn.DeleteTopics(testTopic)
			conn.Close()
		}
	})

	// создаем тестовый HTTP-сервер
	var received []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		received = append(received, buf.String())
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// получаем реальный порт сервера
	u, _ := url.Parse(ts.URL)
	testPort := u.Port()

	// подменяем настройки
	topic = testTopic
	groupID = testGroupID
	os.Setenv("L0_PORT", testPort) // используем порт тестового сервера

	// запускаем консюмер на 15 секунд
	go func() {
		_, cancel := context.WithCancel(context.Background())
		defer cancel()
		time.AfterFunc(15*time.Second, cancel)
		main()
	}()

	// ждём инициализации
	time.Sleep(3 * time.Second)

	// отправляем сообщения
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{brokerAddr},
		Topic:   testTopic,
	})
	defer writer.Close()

	// отправляем сообщения
	for _, msg := range testMessages {
		err := writer.WriteMessages(context.Background(), kafka.Message{
			Value: []byte(msg),
		})
		require.NoError(t, err)
	}

	// проверяем результаты:
	// количество сообщений
	assert.Eventually(t, func() bool {
		return len(received) == len(testMessages)
	}, 15*time.Second, 1*time.Second)

	// соответствие сообщений
	for i := range testMessages {
		assert.JSONEq(t, testMessages[i], received[i])
	}
}
