package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcKafka "github.com/testcontainers/testcontainers-go/modules/kafka"
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

// TestProduceIntegrations тестирует работу продюсера с подключением к тестовому брокеру
func TestProduceIntegrations(t *testing.T) {

	if testing.Short() {
		t.Skip("Пропускаем интеграционный тест в short режиме.")
	}

	// инициализируем noop tracer
	tracer = noop.NewTracerProvider().Tracer("test")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 1. Запускаем контейнер с тестовой кафкой (для дополнительного разнообразия берём образ, отличный от проектного)
	kafkaContainer, err := tcKafka.Run(ctx, "confluentinc/confluent-local:7.5.0",
		tcKafka.WithClusterID("test-cluster"))
	defer func() {
		if err := testcontainers.TerminateContainer(kafkaContainer); err != nil {
			t.Logf("ошибка при остановке kafkaContainer в тесте: %v", err)
		}
	}()

	require.NoError(t, err, "Не удалось запустить Kafka контейнер.")

	// 2. Получаем адрес брокера
	brokers, err := kafkaContainer.Brokers(ctx)
	require.NoError(t, err, "Не удалось получить адрес брокера.")
	require.NotEmpty(t, brokers, "Список брокеров пуст.")

	brokerAddr := brokers[0]
	t.Logf("Kafka запущена на: %s.\n", brokerAddr)

	// 3. Устанавливаем соединение с брокером и создаём топик

	// задаём имя топика
	testTopic := "test-topic-producer"

	conn, err := kafka.DialLeader(ctx, "tcp", brokerAddr, testTopic, 0)
	if err != nil {
		t.Logf("ошибка создания тестового топика кафки в тесте: %v\n", err)
	}
	defer func() {
		// закрываем соединение testProduceConn
		if err := conn.Close(); err != nil {
			t.Logf("ошибка при закрытии соединения c тестовым топиком в тесте: %v\n", err)
		}
	}()
	require.NoError(t, err, "Не удалось создать топик в Kafka.")

	// 4. Настраиваем конфиг для продюсера
	host, portStr, err := net.SplitHostPort(brokerAddr)
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)

	// сохраняем оригинальный конфиг
	origCfg := cfg
	defer func() { cfg = origCfg }()

	messagesCount := 10 // количество сообщений отправителя
	writersCount := 5   // количество отправителей
	// сколько сообщений ожидаем на выходе
	expectedTotalMessages := messagesCount * writersCount

	// меняем значения в конфиге на более соответствующие тестам
	cfg = &ProducerConfig{
		Topic:         testTopic,
		KafkaHost:     host,
		KafkaPort:     port,
		MassagesCount: messagesCount,
		WritersCount:  writersCount,
	}

	t.Logf("Конфиг: %d врайтеров, по %d сообщений каждый.\n", cfg.WritersCount, cfg.MassagesCount)

	// 5. Создаём ряд врайтеров
	writers := make([]*kafka.Writer, cfg.WritersCount)
	for i := 0; i < len(writers); i++ {
		// определяем продюсер
		writers[i] = &kafka.Writer{
			Addr:         kafka.TCP(brokerAddr), // список брокеров
			Topic:        cfg.Topic,             // имя топика, в который будем слать сообщения
			RequiredAcks: kafka.RequireAll,      // максимальный контроль доставки (подтверждение от всех реплик, если бы они были)
			BatchSize:    cfg.MassagesCount,     // отправляем батчем для скорости в тесте
		}
		defer func(idx int) {
			if err := writers[idx].Close(); err != nil {
				log.Printf("Ошибка при закрытии врайтера продюсера в тесте: %v.\n", err)
			}
		}(i)
	}

	var generatedCount int64 // количество сгенерированных сообщений
	var countSended int64    // количество отправленных сообщений
	var failedCount int64    // количество ошибок при отправке сообщений

	t.Logf("Запускаем %d врайтеров.\n", len(writers))

	// 6. Отправляем сообщения
	startTime := time.Now()
	var wg sync.WaitGroup

	for i := 0; i < len(writers); i++ {
		wg.Add(1)
		go sendMessages(ctx, writers[i], &generatedCount, &countSended, &failedCount, &wg)
	}

	wg.Wait()
	sendDuration := time.Since(startTime)
	t.Logf("Отправка завершена за %v.\n", sendDuration)

	// 7. Проверяем результаты отправки
	t.Logf("Сгенерировано: %d, Отправлено: %d, Ошибок: %d.\n", atomic.LoadInt64(&generatedCount), atomic.LoadInt64(&countSended), atomic.LoadInt64(&failedCount))

	// проверяем, что нет ошибок при отправке
	assert.Equal(t, int64(0), atomic.LoadInt64(&failedCount), "Не должно быть ошибок при отправке сообщений.")

	// проверяем, что сгенерировано и отправлено ожидаемое количество
	require.Equal(t, int64(expectedTotalMessages), atomic.LoadInt64(&generatedCount),
		"Количество сгенерированных сообщений должно соответствовать ожидаемому.")
	assert.Equal(t, int64(expectedTotalMessages), atomic.LoadInt64(&countSended),
		"Количество отправленных сообщений должно соответствовать ожидаемому.")

	// 8. Создаём ридер из тестового топика
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{brokerAddr},
		Topic:    cfg.Topic,
		GroupID:  "test-group-producer",
		MinBytes: 10,
		MaxBytes: 10e6,
	})
	defer func() {
		if err := r.Close(); err != nil {
			t.Logf("ошибка при закрытии ридера из тестового топика в тесте: %v", err)
		}
	}()

	// 9. Вычитываем сообщения из кафки
	var messages []kafka.Message
	readCtx, testCancel := context.WithTimeout(ctx, 10*time.Second)
	defer testCancel()

	readStart := time.Now()
	for {
		msg, err := r.FetchMessage(readCtx)
		if err != nil {
			if err == context.DeadlineExceeded || err == context.Canceled {
				break
			}
			t.Logf("ошибка чтения сообщения: %v", err)
			break
		}
		messages = append(messages, msg)
		r.CommitMessages(readCtx, msg)
	}
	readDuration := time.Since(readStart)

	mesCount := len(messages)
	t.Logf("Прочитано %d сообщений за %v.", mesCount, readDuration)

	// 10. Проверяем результаты
	offsetCtx, offsetCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer offsetCancel()

	conn2, err := kafka.DialLeader(offsetCtx, "tcp", brokerAddr, testTopic, 0)
	require.NoError(t, err, "Не удалось подключиться к Kafka для проверки offset.")
	defer func() {
		if err := conn2.Close(); err != nil {
			t.Logf("ошибка закрытия подключения conn2 к кафке в тесте: %v", err)
		}
	}()

	// получаем первый и последний offset
	firstOffset, err := conn2.ReadFirstOffset()
	require.NoError(t, err, "Не удалось получить первый offset.")

	lastOffset, err := conn2.ReadLastOffset()
	require.NoError(t, err, "Не удалось получить последний offset.")

	// количество сообщений в топике
	messagesInTopic := lastOffset - firstOffset
	t.Logf("Смещения в топике: first=%d, last=%d, сообщений=%d.",
		firstOffset, lastOffset, messagesInTopic)

	// проверяем, что количество сообщений в топике соответствует ожидаемому
	require.Equal(t, int64(expectedTotalMessages), messagesInTopic,
		"Количество сообщений в топике должно соответствовать отправленным (offset: %d-%d=%d, ожидали: %d).",
		lastOffset, firstOffset, messagesInTopic, expectedTotalMessages)

	// если прочитали меньше, чем ожидали, но offset показывает правильное количество, значит проблема во времени чтения (таймаут)
	assert.Equal(t, expectedTotalMessages, mesCount,
		"Количество прочитанных сообщений должно соответствовать отправленным. Прочитано: %d, Ожидалось: %d, Offset показывает: %d.",
		mesCount, expectedTotalMessages, messagesInTopic)

	// если прочитали не все, но offset правильный, даём подсказку
	if mesCount < expectedTotalMessages {
		t.Logf("ошибка: прочитано %d из %d сообщений. Попробуйте увеличить таймаут чтения (сейчас 10 сек).\n",
			mesCount, expectedTotalMessages)

		// хотя бы одно сообщение должно быть прочитано, если не прочитано ни одного - что-то не так
		require.Greater(t, mesCount, 0,
			"Должно быть прочитано хотя бы одно сообщение. "+"Проверьте настройки ридера и соединение с Kafka.")
	} else {
		t.Logf("✓ Успешно прочитаны все %d сообщений.", mesCount)
	}

	t.Logf("✅ Тест продюсера завершен: отправлено %d сообщений через %d врайтеров за %v.\n"+
		"В топике: %d сообщений (по offset), прочитано: %d сообщений за %v.\n",
		expectedTotalMessages, writersCount, sendDuration, messagesInTopic, mesCount, readDuration)
}
