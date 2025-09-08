package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/models"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDialLeader тестирует возможность подключения к брокеру и создание топика
func TestDialLeader(t *testing.T) {

	// тестовые варианты
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
			wantErr:    true, // ожидается ли ошибка
		},
		{
			name:       "Рабочий брокер - успешное подключение",
			brokerAddr: "localhost:9092",
			topic:      "my-topic-L0",
			wantErr:    false, // ожидается ли ошибка
		},
	}

	// проверяем по очереди
	for _, tt := range testCases {

		t.Run(tt.name, func(t *testing.T) {

			// создаём подключение с тестовыми параметрами
			conn, err := kafka.DialLeader(context.Background(), "tcp", tt.brokerAddr, tt.topic, 0)

			if tt.wantErr { // если ошибка ожидается
				require.Error(t, err, "Ожидалась ошибка подключения.")
				assert.Nil(t, conn, "Соединение должно быть nil при ошибке.") // убеждаемся, что соединение не было установлено
				return
			}

			// для рабочего брокера
			require.NoError(t, err, "Неожиданная ошибка подключения.") // проверяем отсутствие ошибок
			require.NotNil(t, conn, "Соединение не должно быть nil.")  // проверяем соединение
			defer conn.Close()

			// проверим топик на наличие партиций
			partitions, err := conn.ReadPartitions(tt.topic)
			assert.NoError(t, err, "Ошибка при получении информации о партициях.")
			assert.NotEmpty(t, partitions, "Топик должен содержать минимум одну партицию.")
		})
	}
}

// TestMessageGenerate проверяет работу функции генерации сообщений
func TestMessageGenerate(t *testing.T) {

	// генерируем одно тестовое сообщение
	msgs := messageGenerate(1)

	// проверяем, что функция вернула хотя бы одно сообщение
	require.Len(t, msgs, 1, "Должно быть сгенерировано ровно одно сообщение.")
	require.NotEmpty(t, msgs[0], "Сгенерированное сообщение не должно быть пустым.")

	// десериализуем полученный JSON в структуру
	var order models.Order
	err := json.Unmarshal(msgs[0], &order)

	// основная проверка: корректность структуры JSON (размаршалится или нет)
	assert.NoError(t, err, "Ошибка десериализации JSON. Проверьте соответствие тегов JSON.")

	// проверка связей между моделями
	t.Run("GORM_relationships", func(t *testing.T) {
		// проверка связи Order -> Delivery
		require.NotNil(t, order.Delivery, "Доставка должна быть заполнена.")
		assert.Equal(t, order.ID, order.Delivery.OrderID, "Delivery должен ссылаться на Order.")

		// проверка связи Order -> Payment
		require.NotNil(t, order.Payment, "Платёж должен быть заполнен.")
		assert.Equal(t, order.ID, order.Payment.OrderID, "Payment должен ссылаться на Order.")

		// проверка связи Order -> Items
		require.NotEmpty(t, order.Items, "Список товаров не должен быть пустым.")
		for _, item := range order.Items {
			assert.Equal(t, order.ID, item.OrderID, "Item должен ссылаться на Order.")
		}
	})

	// проверка критически важных полей, без которых весь сервис теряет смысл
	t.Run("Critical_fields", func(t *testing.T) {
		assert.NotEmpty(t, order.OrderUID, "OrderUID: обязательное поле.")
		assert.NotEmpty(t, order.Payment.Transaction, "Payment.Transaction: обязательное поле.")
		assert.Equal(t, order.OrderUID, order.Payment.Transaction, "OrderUID и Payment.Transaction должны совпадать.")
	})
}

// TestWriteMessage тестирует отправку и получение определённого количества сообщений
func TestWriteMessage(t *testing.T) {

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

	// генерируем тестовые данные
	expectedCount := 5 // количество тестовых сообщений
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
