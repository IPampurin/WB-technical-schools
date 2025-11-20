package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDialLeader тестирует возможность подключения к брокеру и создание топика
func TestDialLeader(t *testing.T) {
	testCases := []struct {
		name       string
		brokerAddr string
		topic      string
		wantErr    bool
	}{
		{
			name:       "Фейковый брокер - нет соединения",
			brokerAddr: "invalid.host.never.exists:9092",
			topic:      "test-topic",
			wantErr:    true,
		},
		{
			name:       "Рабочий брокер - успешное подключение",
			brokerAddr: "localhost:9092",
			topic:      "my-topic-L0",
			wantErr:    false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := kafka.DialLeader(context.Background(), "tcp", tt.brokerAddr, tt.topic, 0)

			if tt.wantErr {
				require.Error(t, err, "Ожидалась ошибка подключения.")
				assert.Nil(t, conn, "Соединение должно быть nil при ошибке.")
				return
			}

			require.NoError(t, err, "Неожиданная ошибка подключения.")
			require.NotNil(t, conn, "Соединение не должно быть nil.")
			defer conn.Close()

			partitions, err := conn.ReadPartitions(tt.topic)
			assert.NoError(t, err, "Ошибка при получении информации о партициях.")
			assert.NotEmpty(t, partitions, "Топик должен содержать минимум одну партицию.")
		})
	}
}

// TestMessageGenerate проверяет работу функции генерации сообщений
func TestMessageGenerate(t *testing.T) {
	// генерируем тестовые сообщения
	msgs := messageGenerate(3)

	// проверяем количество
	require.Len(t, msgs, 3, "Должно быть сгенерировано 3 сообщения.")

	for i, msg := range msgs {
		t.Run(fmt.Sprintf("Message_%d", i), func(t *testing.T) {
			require.NotEmpty(t, msg, "Сгенерированное сообщение не должно быть пустым.")

			// десериализуем в структуру продюсера (не models!)
			var order Order
			err := json.Unmarshal(msg, &order)
			assert.NoError(t, err, "Ошибка десериализации JSON.")

			// проверка обязательных полей
			t.Run("Required_fields", func(t *testing.T) {
				assert.NotEmpty(t, order.OrderUID, "OrderUID: обязательное поле.")
				assert.NotEmpty(t, order.TrackNumber, "TrackNumber: обязательное поле.")
				assert.NotEmpty(t, order.Entry, "Entry: обязательное поле.")
				assert.NotEmpty(t, order.Locale, "Locale: обязательное поле.")
				assert.NotEmpty(t, order.CustomerID, "CustomerID: обязательное поле.")
				assert.NotEmpty(t, order.DeliveryService, "DeliveryService: обязательное поле.")
				assert.NotEmpty(t, order.Shardkey, "Shardkey: обязательное поле.")
				assert.Greater(t, order.SMID, 0, "SMID: должен быть больше 0.")
			})

			// проверка Delivery
			t.Run("Delivery", func(t *testing.T) {
				assert.NotEmpty(t, order.Delivery.Name, "Delivery.Name: обязательное поле.")
				assert.NotEmpty(t, order.Delivery.Phone, "Delivery.Phone: обязательное поле.")
				assert.NotEmpty(t, order.Delivery.Zip, "Delivery.Zip: обязательное поле.")
				assert.NotEmpty(t, order.Delivery.City, "Delivery.City: обязательное поле.")
				assert.NotEmpty(t, order.Delivery.Address, "Delivery.Address: обязательное поле.")
				assert.NotEmpty(t, order.Delivery.Region, "Delivery.Region: обязательное поле.")
				assert.NotEmpty(t, order.Delivery.Email, "Delivery.Email: обязательное поле.")
				assert.Contains(t, order.Delivery.Email, "@", "Delivery.Email: должен содержать @")
			})

			// проверка Payment
			t.Run("Payment", func(t *testing.T) {
				assert.NotEmpty(t, order.Payment.Transaction, "Payment.Transaction: обязательное поле.")
				assert.NotEmpty(t, order.Payment.Currency, "Payment.Currency: обязательное поле.")
				assert.NotEmpty(t, order.Payment.Provider, "Payment.Provider: обязательное поле.")
				assert.NotEmpty(t, order.Payment.Bank, "Payment.Bank: обязательное поле.")
				assert.Equal(t, order.OrderUID, order.Payment.Transaction, "OrderUID и Payment.Transaction должны совпадать.")
				assert.Greater(t, order.Payment.Amount, 0.0, "Payment.Amount: должен быть больше 0.")
				assert.NotZero(t, order.Payment.PaymentDT, "Payment.PaymentDT: не должен быть нулевым.")
				// Дополнительная проверка что дата реалистичная (не в будущем)
				assert.LessOrEqual(t, order.Payment.PaymentDT, time.Now().Unix(), "Payment.PaymentDT: не должен быть в будущем")
			})

			// проверка Items
			t.Run("Items", func(t *testing.T) {
				assert.GreaterOrEqual(t, len(order.Items), 1, "Items: должен быть минимум 1 товар.")
				for j, item := range order.Items {
					assert.Greater(t, item.ChrtID, 0, "Item[%d].ChrtID: должен быть больше 0", j)
					assert.NotEmpty(t, item.RID, "Item[%d].RID: обязательное поле", j)
					assert.NotEmpty(t, item.Name, "Item[%d].Name: обязательное поле", j)
					assert.NotEmpty(t, item.Size, "Item[%d].Size: обязательное поле", j)
					assert.NotEmpty(t, item.Brand, "Item[%d].Brand: обязательное поле", j)
					assert.Greater(t, item.Price, 0.0, "Item[%d].Price: должен быть больше 0", j)
					assert.Greater(t, item.TotalPrice, 0.0, "Item[%d].TotalPrice: должен быть больше 0", j)
					assert.Greater(t, item.NMID, 0, "Item[%d].NMID: должен быть больше 0", j)
					assert.GreaterOrEqual(t, item.Status, 0, "Item[%d].Status: должен быть неотрицательным", j)
				}
			})
		})
	}
}

// TestMessageGenerateCount проверяет корректность количества сообщений
func TestMessageGenerateCount(t *testing.T) {
	testCases := []struct {
		count int
	}{
		{count: 0},
		{count: 1},
		{count: 5},
		{count: 10},
	}

	for _, tt := range testCases {
		t.Run(fmt.Sprintf("Count_%d", tt.count), func(t *testing.T) {
			msgs := messageGenerate(tt.count)
			assert.Len(t, msgs, tt.count, "Количество сообщений должно соответствовать запрошенному")
		})
	}
}

// TestWriteMessage тестирует отправку и получение сообщений
func TestWriteMessage(t *testing.T) {
	// задаём начальные условия
	testTopic := "test-topic-L0"
	brokerAddr := "localhost:9092"

	// подключаемся к брокеру
	conn, err := kafka.DialLeader(context.Background(), "tcp", brokerAddr, testTopic, 0)
	require.NoError(t, err, "Не удалось подключиться к брокеру.")
	defer conn.Close()

	// готовим продюсер
	writer := &kafka.Writer{
		Addr:  kafka.TCP(brokerAddr),
		Topic: testTopic,
	}
	defer writer.Close()

	// генерируем тестовые данные
	expectedCount := 3
	messages := messageGenerate(expectedCount)
	require.Len(t, messages, expectedCount, "Количество сгенерированных сообщений не соответствует ожидаемому.")

	// отправляем сообщения
	for i, msg := range messages {
		err := writer.WriteMessages(context.Background(), kafka.Message{
			Key:   []byte(fmt.Sprintf("key-%d", i)),
			Value: msg,
		})
		require.NoError(t, err, "Ошибка при отправке сообщения %d.", i+1)
	}

	// проверяем доставку - вычитываем сообщения
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{brokerAddr},
		Topic:    testTopic,
		GroupID:  "test-group",
		MinBytes: 1,
		MaxBytes: 10e6, // 10MB
		MaxWait:  5 * time.Second,
	})
	defer reader.Close()

	// вычитываем и проверяем сообщения
	counter := 0
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for counter < expectedCount {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if err == context.DeadlineExceeded {
				break
			}
			require.NoError(t, err, "Ошибка при чтении сообщения.")
		}

		// проверяем структуру полученного сообщения
		var order Order
		err = json.Unmarshal(msg.Value, &order)
		assert.NoError(t, err, "Ошибка десериализации полученного сообщения")
		assert.NotEmpty(t, order.OrderUID, "Полученный OrderUID не должен быть пустым")

		counter++
	}

	assert.Equal(t, expectedCount, counter, "Количество прочитанных сообщений не совпадает.")
}
