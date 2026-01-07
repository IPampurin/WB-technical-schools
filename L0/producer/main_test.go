package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

// TestGetEnvString тестирует извлечение строковой переменной окружения
func TestGetEnvString(t *testing.T) {
	// Сохраняем оригинальное значение переменной
	const testEnvVar = "TEST_ENV_VAR_GETENVSTRING"
	originalValue, existed := os.LookupEnv(testEnvVar)
	defer func() {
		if existed {
			os.Setenv(testEnvVar, originalValue)
		} else {
			os.Unsetenv(testEnvVar)
		}
	}()

	tests := []struct {
		name         string
		setValue     string
		defaultValue string
		expected     string
	}{
		{
			name:         "переменная не установлена",
			setValue:     "",
			defaultValue: "default_value",
			expected:     "default_value",
		},
		{
			name:         "переменная установлена",
			setValue:     "custom_value",
			defaultValue: "default_value",
			expected:     "custom_value",
		},
		{
			name:         "пустая строка как значение",
			setValue:     "",
			defaultValue: "default_value",
			expected:     "",
		},
		{
			name:         "невалидное значение",
			setValue:     "1",
			defaultValue: "int",
			expected:     "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем или сбрасываем переменную окружения
			if tt.setValue == "" && tt.name != "пустая строка как значение" {
				os.Unsetenv(testEnvVar)
			} else {
				os.Setenv(testEnvVar, tt.setValue)
			}

			// Вызываем тестируемую функцию
			result := getEnvString(testEnvVar, tt.defaultValue)

			// Проверяем результат
			assert.Equal(t, tt.expected, result,
				"Для теста '%s': ожидалось '%s', получено '%s'",
				tt.name, tt.expected, result)
		})
	}
}

// TestGetEnvInt тестирует извлечение числовой переменной окружения
func TestGetEnvInt(t *testing.T) {
	// Сохраняем оригинальное значение переменной
	const testEnvVar = "TEST_ENV_VAR_GETENVINT"
	originalValue, existed := os.LookupEnv(testEnvVar)
	defer func() {
		if existed {
			os.Setenv(testEnvVar, originalValue)
		} else {
			os.Unsetenv(testEnvVar)
		}
	}()

	tests := []struct {
		name         string
		setValue     string
		defaultValue int
		expected     int
	}{
		{
			name:         "переменная не установлена",
			setValue:     "",
			defaultValue: 100,
			expected:     100,
		},
		{
			name:         "переменная установлена корректно",
			setValue:     "200",
			defaultValue: 100,
			expected:     200,
		},
		{
			name:         "пустая строка как значение",
			setValue:     "",
			defaultValue: 100,
			expected:     100,
		},
		{
			name:         "отрицательное число",
			setValue:     "-50",
			defaultValue: 100,
			expected:     -50,
		},
		{
			name:         "ноль",
			setValue:     "0",
			defaultValue: 100,
			expected:     0,
		},
		{
			name:         "невалидное значение (строка)",
			setValue:     "abc",
			defaultValue: 100,
			expected:     100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем или сбрасываем переменную окружения
			if tt.setValue == "" && tt.name != "пустая строка как значение" {
				os.Unsetenv(testEnvVar)
			} else {
				os.Setenv(testEnvVar, tt.setValue)
			}

			// Вызываем тестируемую функцию
			result := getEnvInt(testEnvVar, tt.defaultValue)

			// Проверяем результат
			assert.Equal(t, tt.expected, result,
				"Для теста '%s': ожидалось '%d', получено '%d'",
				tt.name, tt.expected, result)
		})
	}
}

// TestReadConfigProducer тестирует получение конфигурации продюсера
func TestReadConfigProducer(t *testing.T) {
	// Сохраняем оригинальные значения всех переменных окружения
	envVars := map[string]string{
		"TOPIC_NAME_STR": "",
		"KAFKA_PORT_NUM": "",
		"MESSAGES_COUNT": "",
		"WRITERS_COUNT":  "",
	}

	// Сохраняем и очищаем переменные
	for envVar := range envVars {
		if val, exists := os.LookupEnv(envVar); exists {
			envVars[envVar] = val
			defer os.Setenv(envVar, val)
		} else {
			defer os.Unsetenv(envVar)
		}
		os.Unsetenv(envVar)
	}

	tests := []struct {
		name     string
		setEnv   map[string]string
		expected *ProducerConfig
	}{
		{
			name:   "значения по умолчанию",
			setEnv: map[string]string{}, // все переменные не установлены
			expected: &ProducerConfig{
				Topic:         topicNameConst,
				KafkaPort:     kafkaPortConst,
				MassagesCount: massagesCountConst,
				WritersCount:  writersCountConst,
			},
		},
		{
			name: "частично переопределенные значения",
			setEnv: map[string]string{
				"TOPIC_NAME_STR": "custom-topic",
				"MESSAGES_COUNT": "100",
				"WRITERS_COUNT":  "10",
			},
			expected: &ProducerConfig{
				Topic:         "custom-topic",
				KafkaPort:     kafkaPortConst, // остается по умолчанию
				MassagesCount: 100,
				WritersCount:  10,
			},
		},
		{
			name: "все значения переопределены",
			setEnv: map[string]string{
				"TOPIC_NAME_STR": "test-topic",
				"KAFKA_PORT_NUM": "9093",
				"MESSAGES_COUNT": "50",
				"WRITERS_COUNT":  "5",
			},
			expected: &ProducerConfig{
				Topic:         "test-topic",
				KafkaPort:     9093,
				MassagesCount: 50,
				WritersCount:  5,
			},
		},
		{
			name: "некорректные числовые значения",
			setEnv: map[string]string{
				"KAFKA_PORT_NUM": "not-a-number",
				"MESSAGES_COUNT": "invalid",
				"WRITERS_COUNT":  "abc",
			},
			expected: &ProducerConfig{
				Topic:         topicNameConst,
				KafkaPort:     kafkaPortConst, // значение по умолчанию при ошибке парсинга
				MassagesCount: massagesCountConst,
				WritersCount:  writersCountConst,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем переменные окружения для теста
			for envVar, value := range tt.setEnv {
				os.Setenv(envVar, value)
			}

			// Вызываем тестируемую функцию
			config := readConfig()

			// Проверяем результат
			assert.Equal(t, tt.expected.Topic, config.Topic, "Topic не совпадает")
			assert.Equal(t, tt.expected.KafkaPort, config.KafkaPort, "KafkaPort не совпадает")
			assert.Equal(t, tt.expected.MassagesCount, config.MassagesCount, "MassagesCount не совпадает")
			assert.Equal(t, tt.expected.WritersCount, config.WritersCount, "WritersCount не совпадает")

			// Очищаем переменные для следующего теста
			for envVar := range tt.setEnv {
				os.Unsetenv(envVar)
			}
		})
	}
}

// TestCreateDelivery проверяет генерацию структуры Delivery
func TestCreateDelivery(t *testing.T) {
	delivery := createDelivery()

	// Проверяем обязательные поля
	assert.NotEmpty(t, delivery.Name, "Name: обязательное поле")
	assert.NotEmpty(t, delivery.Phone, "Phone: обязательное поле")
	assert.NotEmpty(t, delivery.Zip, "Zip: обязательное поле")
	assert.NotEmpty(t, delivery.City, "City: обязательное поле")
	assert.NotEmpty(t, delivery.Address, "Address: обязательное поле")
	assert.NotEmpty(t, delivery.Region, "Region: обязательное поле")
	assert.NotEmpty(t, delivery.Email, "Email: обязательное поле")

	// Проверяем формат email
	assert.Contains(t, delivery.Email, "@", "Email должен содержать @")
	assert.Contains(t, delivery.Email, ".", "Email должен содержать точку")

	// Проверяем формат телефона (хотя бы содержит цифры)
	hasDigit := false
	for _, r := range delivery.Phone {
		if r >= '0' && r <= '9' {
			hasDigit = true
			break
		}
	}
	assert.True(t, hasDigit, "Phone должен содержать цифры")

	// Проверяем что адрес содержит и улицу и номер
	assert.Contains(t, delivery.Address, ",", "Address должен содержать запятую (улица, номер)")

	// Проверяем длину полей
	assert.GreaterOrEqual(t, len(delivery.Name), 2, "Name должен быть не короче 2 символов")
	assert.GreaterOrEqual(t, len(delivery.City), 2, "City должен быть не короче 2 символов")
}

// TestCreatePayment проверяет генерацию структуры Payment
func TestCreatePayment(t *testing.T) {
	payment := createPayment()

	// Проверяем обязательные поля
	assert.NotEmpty(t, payment.Transaction, "Transaction: обязательное поле")
	assert.NotEmpty(t, payment.Currency, "Currency: обязательное поле")
	assert.NotEmpty(t, payment.Provider, "Provider: обязательное поле")
	assert.NotEmpty(t, payment.Bank, "Bank: обязательное поле")

	// Проверяем UUID формат (хотя бы содержит дефисы)
	assert.Contains(t, payment.Transaction, "-", "Transaction должен быть в формате UUID")
	if payment.RequestID != "" {
		assert.Contains(t, payment.RequestID, "-", "RequestID должен быть в формате UUID")
	}

	// Проверяем числовые поля
	assert.Greater(t, payment.Amount, 0.0, "Amount должен быть больше 0")
	assert.GreaterOrEqual(t, payment.DeliveryCost, 0.0, "DeliveryCost должен быть неотрицательным")
	assert.Greater(t, payment.GoodsTotal, 0.0, "GoodsTotal должен быть больше 0")
	assert.GreaterOrEqual(t, payment.CustomFee, 0.0, "CustomFee должен быть неотрицательным")

	// Проверяем дату (не в будущем и не слишком давно)
	assert.NotZero(t, payment.PaymentDT, "PaymentDT не должен быть нулевым")
	assert.LessOrEqual(t, payment.PaymentDT, time.Now().Unix(), "PaymentDT не должен быть в будущем")
	assert.Greater(t, payment.PaymentDT, time.Now().Add(-365*24*time.Hour).Unix(), "PaymentDT не должен быть старше года")

	// Проверяем логику: сумма = товары + доставка (с учетом округления)
	expectedAmount := payment.GoodsTotal + payment.DeliveryCost
	assert.InDelta(t, expectedAmount, payment.Amount, 0.01, "Amount должен равняться GoodsTotal + DeliveryCost")
}

// TestCreateItem проверяет генерацию структуры Item
func TestCreateItem(t *testing.T) {
	item := createItem()

	// Проверяем обязательные поля
	assert.Greater(t, item.ChrtID, 0, "ChrtID должен быть больше 0")
	assert.NotEmpty(t, item.RID, "RID: обязательное поле")
	assert.NotEmpty(t, item.Name, "Name: обязательное поле")
	assert.NotEmpty(t, item.Size, "Size: обязательное поле")
	assert.NotEmpty(t, item.Brand, "Brand: обязательное поле")

	// Проверяем TrackNumber
	assert.Equal(t, "WBILMTESTTRACK", item.TrackNumber, "TrackNumber должен быть 'WBILMTESTTRACK'")

	// Проверяем числовые поля
	assert.Greater(t, item.Price, 0.0, "Price должен быть больше 0")
	assert.GreaterOrEqual(t, item.Sale, 0.0, "Sale должен быть неотрицательным")
	assert.LessOrEqual(t, item.Sale, 100.0, "Sale не должен превышать 100%")
	assert.Greater(t, item.TotalPrice, 0.0, "TotalPrice должен быть больше 0")
	assert.Greater(t, item.NMID, 0, "NMID должен быть больше 0")
	assert.GreaterOrEqual(t, item.Status, 0, "Status должен быть неотрицательным")

	// Проверяем логику: TotalPrice = Price * (1 - Sale/100) (с учетом округления)
	expectedTotalPrice := item.Price * (1 - item.Sale/100)
	assert.InDelta(t, expectedTotalPrice, item.TotalPrice, 0.01,
		"TotalPrice должен равняться Price * (1 - Sale/100)")

	// Проверяем UUID формат для RID
	assert.Contains(t, item.RID, "-", "RID должен быть в формате UUID")

	// Проверяем размер (допустимые значения)
	validSizes := map[string]bool{"S": true, "M": true, "L": true, "XL": true}
	assert.True(t, validSizes[item.Size], "Size должен быть одним из: S, M, L, XL")

	// Проверяем статус (допустимые значения HTTP)
	assert.GreaterOrEqual(t, item.Status, 200, "Status должен быть >= 200 (HTTP код)")
	assert.LessOrEqual(t, item.Status, 599, "Status должен быть <= 599 (HTTP код)")
}

// TestMessageGenerate проверяет работу функции генерации сообщений
func TestMessageGenerate(t *testing.T) {

	// инициализируем noop tracer для тестов
	tracer = noop.NewTracerProvider().Tracer("test")

	// генерируем тестовые сообщения
	testCount := 3
	msgs := messageGenerate(testCount)

	// проверяем количество
	require.Len(t, msgs, testCount, "Должно быть сгенерировано %d сообщений.", testCount)

	for i, msg := range msgs {
		t.Run(fmt.Sprintf("Message_%d", i), func(t *testing.T) {
			require.NotNil(t, msg, "Сгенерированное сообщение не должно быть nil.")
			require.NotEmpty(t, msg, "Сгенерированное сообщение не должно быть пустым.")

			// десериализуем в структуру продюсера (не models!)
			var order Order
			err := json.Unmarshal(msg, &order)
			assert.NoError(t, err, "Ошибка десериализации JSON: %v", err)

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

	// инициализируем noop tracer для тестов
	tracer = noop.NewTracerProvider().Tracer("test")

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

			// Для ненулевого количества проверяем что сообщения не пустые
			if tt.count > 0 {
				for i, msg := range msgs {
					assert.NotEmpty(t, msg, "Сообщение %d не должно быть пустым", i)
				}
			}
		})
	}
}

// TestDialLeader тестирует возможность подключения к брокеру и создание топика
func TestDialLeader(t *testing.T) {
	// Mock тест для проверки логики без реального подключения к Kafka
	testCases := []struct {
		name       string
		brokerAddr string
		topic      string
		wantErr    bool
		skip       bool // флаг для пропуска тестов требующих реального Kafka
	}{
		{
			name:       "Некорректный адрес брокера",
			brokerAddr: "invalid.host.never.exists:9092",
			topic:      "test-topic",
			wantErr:    true,
			skip:       false, // можно запускать без Kafka
		},
		{
			name:       "Пустой адрес брокера",
			brokerAddr: "",
			topic:      "test-topic",
			wantErr:    true,
			skip:       false,
		},
		{
			name:       "Проверка создания подключения (требует Kafka)",
			brokerAddr: "localhost:9092",
			topic:      "test-topic-integration",
			wantErr:    false,
			skip:       true, // требует реального Kafka
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {

			if tt.skip && testing.Short() {
				t.Skipf("Пропускаем интеграционный тест '%s' в short режиме", tt.name)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			conn, err := kafka.DialLeader(ctx, "tcp", tt.brokerAddr, tt.topic, 0)

			if tt.wantErr {
				assert.Error(t, err, "Ожидалась ошибка подключения для теста '%s'", tt.name)
				assert.Nil(t, conn, "Соединение должно быть nil при ошибке")
				return
			}

			if err != nil {
				t.Logf("Не удалось подключиться к Kafka для теста '%s': %v", tt.name, err)
				t.Skipf("Пропускаем тест '%s' из-за отсутствия подключения к Kafka", tt.name)
				return
			}

			require.NoError(t, err, "Неожиданная ошибка подключения для теста '%s'", tt.name)
			require.NotNil(t, conn, "Соединение не должно быть nil")

			defer func() {
				if conn != nil {
					conn.Close()
				}
			}()

			// проверяем что соединение работает
			partitions, err := conn.ReadPartitions()
			assert.NoError(t, err, "Ошибка при получении информации о партициях")
			assert.NotNil(t, partitions, "Список партиций не должен быть nil")
		})
	}
}

// TestWriteMessage тестирует отправку и получение сообщений
func TestWriteMessage(t *testing.T) {

	if testing.Short() {
		t.Skip("Пропускаем интеграционный тест с Kafka в short режиме")
	}

	// инициализируем noop tracer для тестов
	tracer = noop.NewTracerProvider().Tracer("test")

	// проверяем доступность Kafka перед запуском теста
	if !isKafkaAvailable("localhost:9092") {
		t.Skip("Kafka недоступна на localhost:9092, пропускаем тест")
	}

	// задаём начальные условия
	testTopic := "test-topic-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	brokerAddr := "localhost:9092"

	// подключаемся к брокеру и создаем топик
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// подключаемся к брокеру и создаем топик
	conn, err := kafka.DialLeader(ctx, "tcp", brokerAddr, testTopic, 0)
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
		GroupID:  "test-group-" + testTopic,
		MinBytes: 1,
		MaxBytes: 10e6, // 10MB
		MaxWait:  5 * time.Second,
	})
	defer reader.Close()

	// вычитываем и проверяем сообщения
	counter := 0
	readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer readCancel()

	for counter < expectedCount {
		msg, err := reader.FetchMessage(readCtx)
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

		// подтверждаем прочтение сообщения
		err = reader.CommitMessages(context.Background(), msg)
		assert.NoError(t, err, "Ошибка при подтверждении сообщения")

		counter++
	}

	assert.Equal(t, expectedCount, counter, "Количество прочитанных сообщений не совпадает.")

	// очистка тестового топика
	t.Cleanup(func() {
		conn, err := kafka.Dial("tcp", brokerAddr)
		if err == nil {
			conn.DeleteTopics(testTopic)
			conn.Close()
		}
	})
}

// isKafkaAvailable проверяет доступность Kafka
func isKafkaAvailable(brokerAddr string) bool {

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, err := kafka.DialContext(ctx, "tcp", brokerAddr)
	if err != nil {
		return false
	}
	defer conn.Close()

	return true
}
