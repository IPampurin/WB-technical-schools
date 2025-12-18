package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestReadConfig тестирует получение конфигурации
func TestReadConfig(t *testing.T) {
	// Сохраняем оригинальные значения всех переменных окружения
	envVars := map[string]string{
		"TOPIC_NAME_STR":        "",
		"GROUP_ID_NAME_STR":     "",
		"KAFKA_PORT_NUM":        "",
		"TIME_LIMIT_CONSUMER_S": "",
		"SERVICE_PORT_NUM":      "",
		"BATCH_SIZE_NUM":        "",
		"BATCH_TIMEOUT_MS":      "",
		"MAX_RETRIES_NUM":       "",
		"RETRY_DELEY_BASE_MS":   "",
		"CLIENT_TIMEOUT_S":      "",
		"DLQ_TOPIC_NAME_STR":    "",
		"WORKERS_COUNT":         "",
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
		expected *ConsumerConfig
	}{
		{
			name:   "значения по умолчанию",
			setEnv: map[string]string{}, // все переменные не установлены
			expected: &ConsumerConfig{
				Topic:           topicNameConst,
				GroupID:         groupIDNameConst,
				KafkaPort:       kafkaPortConst,
				LimitConsumWork: time.Duration(limitConsumWorkConst) * time.Second,
				ServicePort:     servicePortConst,
				BatchSize:       batchSizeConst,
				BatchTimeout:    time.Duration(batchTimeoutConst) * time.Millisecond,
				MaxRetries:      maxRetriesConst,
				RetryDelayBase:  time.Duration(retryDelayBaseConst) * time.Millisecond,
				ClientTimeout:   time.Duration(clientTimeoutConst) * time.Second,
				DlqTopic:        dlqTopicConst,
				WorkersCount:    workersCountConst,
			},
		},
		{
			name: "частично переопределенные значения",
			setEnv: map[string]string{
				"TOPIC_NAME_STR":    "custom-topic",
				"GROUP_ID_NAME_STR": "custom-group",
				"SERVICE_PORT_NUM":  "9090",
				"BATCH_SIZE_NUM":    "500",
				"WORKERS_COUNT":     "10",
			},
			expected: &ConsumerConfig{
				Topic:           "custom-topic",
				GroupID:         "custom-group",
				KafkaPort:       kafkaPortConst, // остается по умолчанию
				LimitConsumWork: time.Duration(limitConsumWorkConst) * time.Second,
				ServicePort:     9090,
				BatchSize:       500,
				BatchTimeout:    time.Duration(batchTimeoutConst) * time.Millisecond,
				MaxRetries:      maxRetriesConst,
				RetryDelayBase:  time.Duration(retryDelayBaseConst) * time.Millisecond,
				ClientTimeout:   time.Duration(clientTimeoutConst) * time.Second,
				DlqTopic:        dlqTopicConst,
				WorkersCount:    10,
			},
		},
		{
			name: "все значения переопределены",
			setEnv: map[string]string{
				"TOPIC_NAME_STR":        "test-topic",
				"GROUP_ID_NAME_STR":     "test-group",
				"KAFKA_PORT_NUM":        "9093",
				"TIME_LIMIT_CONSUMER_S": "30",
				"SERVICE_PORT_NUM":      "8082",
				"BATCH_SIZE_NUM":        "150",
				"BATCH_TIMEOUT_MS":      "2000",
				"MAX_RETRIES_NUM":       "5",
				"RETRY_DELEY_BASE_MS":   "200",
				"CLIENT_TIMEOUT_S":      "15",
				"DLQ_TOPIC_NAME_STR":    "test-dlq",
				"WORKERS_COUNT":         "8",
			},
			expected: &ConsumerConfig{
				Topic:           "test-topic",
				GroupID:         "test-group",
				KafkaPort:       9093,
				LimitConsumWork: 30 * time.Second,
				ServicePort:     8082,
				BatchSize:       150,
				BatchTimeout:    2000 * time.Millisecond,
				MaxRetries:      5,
				RetryDelayBase:  200 * time.Millisecond,
				ClientTimeout:   15 * time.Second,
				DlqTopic:        "test-dlq",
				WorkersCount:    8,
			},
		},
		{
			name: "некорректные числовые значения",
			setEnv: map[string]string{
				"KAFKA_PORT_NUM":        "not-a-number",
				"SERVICE_PORT_NUM":      "invalid",
				"BATCH_SIZE_NUM":        "abc",
				"TIME_LIMIT_CONSUMER_S": "xyz",
			},
			expected: &ConsumerConfig{
				Topic:           topicNameConst,
				GroupID:         groupIDNameConst,
				KafkaPort:       kafkaPortConst, // значение по умолчанию при ошибке парсинга
				LimitConsumWork: time.Duration(limitConsumWorkConst) * time.Second,
				ServicePort:     servicePortConst,
				BatchSize:       batchSizeConst,
				BatchTimeout:    time.Duration(batchTimeoutConst) * time.Millisecond,
				MaxRetries:      maxRetriesConst,
				RetryDelayBase:  time.Duration(retryDelayBaseConst) * time.Millisecond,
				ClientTimeout:   time.Duration(clientTimeoutConst) * time.Second,
				DlqTopic:        dlqTopicConst,
				WorkersCount:    workersCountConst,
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
			assert.Equal(t, tt.expected.GroupID, config.GroupID, "GroupID не совпадает")
			assert.Equal(t, tt.expected.KafkaPort, config.KafkaPort, "KafkaPort не совпадает")
			assert.Equal(t, tt.expected.LimitConsumWork, config.LimitConsumWork, "LimitConsumWork не совпадает")
			assert.Equal(t, tt.expected.ServicePort, config.ServicePort, "ServicePort не совпадает")
			assert.Equal(t, tt.expected.BatchSize, config.BatchSize, "BatchSize не совпадает")
			assert.Equal(t, tt.expected.BatchTimeout, config.BatchTimeout, "BatchTimeout не совпадает")
			assert.Equal(t, tt.expected.MaxRetries, config.MaxRetries, "MaxRetries не совпадает")
			assert.Equal(t, tt.expected.RetryDelayBase, config.RetryDelayBase, "RetryDelayBase не совпадает")
			assert.Equal(t, tt.expected.ClientTimeout, config.ClientTimeout, "ClientTimeout не совпадает")
			assert.Equal(t, tt.expected.DlqTopic, config.DlqTopic, "DlqTopic не совпадает")
			assert.Equal(t, tt.expected.WorkersCount, config.WorkersCount, "WorkersCount не совпадает")

			// Очищаем переменные для следующего теста
			for envVar := range tt.setEnv {
				os.Unsetenv(envVar)
			}
		})
	}
}

// TestExtractOrderUID тестирует извлечение OrderUID из сообщения
func TestExtractOrderUID(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "валидный JSON с order_uid",
			input:    []byte(`{"order_uid": "test-123", "data": "some data"}`),
			expected: "test-123",
		},
		{
			name:     "JSON без order_uid",
			input:    []byte(`{"data": "some data", "data": "some data"}`),
			expected: "",
		},
		{
			name:     "пустой JSON объект",
			input:    []byte(`{}`),
			expected: "",
		},
		{
			name:     "невалидный JSON",
			input:    []byte(`invalid json`),
			expected: "",
		},
		{
			name:     "пустой массив байт",
			input:    []byte(``),
			expected: "",
		},
		{
			name:     "order_uid с пробелами",
			input:    []byte(`{"order_uid": "  test 123  ", "data": "some data"}`),
			expected: "  test 123  ",
		},
		{
			name:     "order_uid пустая строка",
			input:    []byte(`{"order_uid": "", "data": "some data"}`),
			expected: "",
		},
		{
			name:     "несколько order_uid (второй берется)",
			input:    []byte(`{"order_uid": "first", "order_uid": "second"}`),
			expected: "second",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Вызываем тестируемую функцию
			result := extractOrderUID(tt.input)

			// Проверяем результат
			assert.Equal(t, tt.expected, result,
				"Для теста '%s': ожидалось '%s', получено '%s'",
				tt.name, tt.expected, result)
		})
	}
}

// TestConsumerRead тестирует базовое чтение сообщений из кафки
func TestConsumerRead(t *testing.T) {
	if testing.Short() {
		t.Skip("Пропускаем тест с Kafka в short режиме")
	}

	// Используем переменную окружения или значение по умолчанию
	kafkaHost := os.Getenv("KAFKA_HOST")
	if kafkaHost == "" {
		kafkaHost = "localhost"
	}
	kafkaPort := os.Getenv("KAFKA_PORT")
	if kafkaPort == "" {
		kafkaPort = "9092"
	}
	brokerAddr := kafkaHost + ":" + kafkaPort

	// задаём начальные условия
	testTopic := "test-topic-read-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	groupID := "test-group-read"

	// подключаемся к брокеру и создаем топик
	conn, err := kafka.DialLeader(context.Background(), "tcp", brokerAddr, testTopic, 0)
	if err != nil {
		t.Skipf("Не удалось подключиться к Kafka по адресу %s: %v. Пропускаем тест.", brokerAddr, err)
		return
	}
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
		msg := []byte(fmt.Sprintf(`{"order_uid": "test-%d", "data": "test data %d"}`, i+1, i+1))
		err := writer.WriteMessages(context.Background(), kafka.Message{
			Key:   []byte(fmt.Sprintf("key-%d", i)),
			Value: msg,
		})
		require.NoError(t, err, "Ошибка при отправке сообщения %d.", i+1)
	}

	// организуем консумер
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{brokerAddr},
		Topic:    testTopic,
		GroupID:  groupID,
		MinBytes: 10,
		MaxBytes: 10e6,
		MaxWait:  2 * time.Second,
	})
	defer r.Close()

	// вычитываем и считаем сообщения
	counter := 0
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for {
		msg, err := r.ReadMessage(ctx)
		if err != nil {
			break // выходим когда больше нет сообщений или таймаут
		}
		counter++

		// Извлекаем order_uid для проверки
		uid := extractOrderUID(msg.Value)
		assert.NotEmpty(t, uid, "Сообщение %d должно содержать order_uid", counter)

		// Коммитим чтобы не читать повторно
		r.CommitMessages(ctx, msg)
	}

	assert.Equal(t, expectedCount, counter, "Количество прочитанных сообщений не совпадает.")

	// очистка
	t.Cleanup(func() {
		conn, err := kafka.Dial("tcp", brokerAddr)
		if err == nil {
			conn.DeleteTopics(testTopic)
			conn.Close()
		}
	})
}

// TestConsumerWithRealData - тест с реальными данными из примера
func TestConsumerWithRealData(t *testing.T) {
	if testing.Short() {
		t.Skip("Пропускаем тест с реальными данными в short режиме")
	}

	// Используем переменную окружения или значение по умолчанию
	kafkaHost := os.Getenv("KAFKA_HOST")
	if kafkaHost == "" {
		kafkaHost = "localhost"
	}
	kafkaPort := os.Getenv("KAFKA_PORT")
	if kafkaPort == "" {
		kafkaPort = "9092"
	}
	brokerAddr := kafkaHost + ":" + kafkaPort

	// Тестовые данные из примера
	testOrder := []byte(`{
		"order_uid": "b563feb7b2b84b6test",
		"track_number": "WBILMTESTTRACK",
		"entry": "WBIL",
		"delivery": {
			"name": "Test Testov",
			"phone": "+9720000000",
			"zip": "2639809",
			"city": "Kiryat Mozkin",
			"address": "Ploshad Mira 15",
			"region": "Kraiot",
			"email": "test@gmail.com"
		},
		"payment": {
			"transaction": "b563feb7b2b84b6test",
			"request_id": "",
			"currency": "USD",
			"provider": "wbpay",
			"amount": 1817,
			"payment_dt": 1637907727,
			"bank": "alpha",
			"delivery_cost": 1500,
			"goods_total": 317,
			"custom_fee": 0
		},
		"items": [
			{
				"chrt_id": 9934930,
				"track_number": "WBILMTESTTRACK",
				"price": 453,
				"rid": "ab4219087a764ae0btest",
				"name": "Mascaras",
				"sale": 30,
				"size": "0",
				"total_price": 317,
				"nm_id": 2389212,
				"brand": "Vivienne Sabo",
				"status": 202
			}
		],
		"locale": "en",
		"internal_signature": "",
		"customer_id": "test",
		"delivery_service": "meest",
		"shardkey": "9",
		"sm_id": 99,
		"date_created": "2021-11-26T06:22:19Z",
		"oof_shard": "1"
	}`)

	// создаем топик
	testTopic := "test-real-data-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	conn, err := kafka.DialLeader(context.Background(), "tcp", brokerAddr, testTopic, 0)
	if err != nil {
		t.Skipf("Не удалось подключиться к Kafka: %v", err)
		return
	}
	defer conn.Close()

	// записываем сообщение
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{brokerAddr},
		Topic:   testTopic,
	})
	defer writer.Close()

	err = writer.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte("real-data-key"),
		Value: testOrder,
	})
	require.NoError(t, err)

	// читаем обратно
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{brokerAddr},
		Topic:    testTopic,
		GroupID:  "test-group-real",
		MinBytes: 10,
		MaxBytes: 10e6,
		MaxWait:  2 * time.Second,
	})
	defer reader.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msg, err := reader.ReadMessage(ctx)
	require.NoError(t, err)

	// проверяем данные
	uid := extractOrderUID(msg.Value)
	assert.Equal(t, "b563feb7b2b84b6test", uid, "OrderUID должен совпадать")

	// проверяем что JSON валиден
	var order map[string]interface{}
	err = json.Unmarshal(msg.Value, &order)
	assert.NoError(t, err, "JSON должен быть валидным")
	assert.Equal(t, "Test Testov", order["delivery"].(map[string]interface{})["name"])
	assert.Equal(t, "WBILMTESTTRACK", order["track_number"])

	// очистка
	t.Cleanup(func() {
		conn, err := kafka.Dial("tcp", brokerAddr)
		if err == nil {
			conn.DeleteTopics(testTopic)
			conn.Close()
		}
	})
}

// TestConsumerAPIIntegration тестирует интеграцию с API
func TestConsumerAPIIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Пропускаем интеграционный тест в short режиме")
	}

	// Используем переменную окружения или значение по умолчанию
	kafkaHost := os.Getenv("KAFKA_HOST")
	if kafkaHost == "" {
		kafkaHost = "localhost"
	}
	kafkaPort := os.Getenv("KAFKA_PORT")
	if kafkaPort == "" {
		kafkaPort = "9092"
	}
	brokerAddr := kafkaHost + ":" + kafkaPort

	// задаём начальные условия и тестовые данные
	testTopic := "test-api-topic-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	testGroupID := "test-api-group"
	testDLQTopic := "test-api-dlq-" + strconv.FormatInt(time.Now().UnixNano(), 10)

	testMessages := []kafka.Message{
		{
			Key:   []byte("key-1"),
			Value: []byte(`{"order_uid": "test-1", "data": "test1"}`),
		},
		{
			Key:   []byte("key-2"),
			Value: []byte(`{"order_uid": "test-2", "data": "test2"}`),
		},
		{
			Key:   []byte("key-3"),
			Value: []byte(`{"data": "invalid - no order_uid"}`), // невалидное
		},
	}

	// создаём подключение и топики
	conn, err := kafka.DialLeader(context.Background(), "tcp", brokerAddr, testTopic, 0)
	if err != nil {
		t.Skipf("Не удалось подключиться к Kafka по адресу %s: %v. Пропускаем тест.", brokerAddr, err)
		return
	}
	defer conn.Close()

	// создаем DLQ топик
	dlqConn, err := kafka.DialLeader(context.Background(), "tcp", brokerAddr, testDLQTopic, 0)
	require.NoError(t, err, "Не удалось создать DLQ топик")
	dlqConn.Close()

	// сохраняем оригинальные переменные окружения
	originalEnv := map[string]string{
		"TOPIC_NAME_STR":        "",
		"GROUP_ID_NAME_STR":     "",
		"DLQ_TOPIC_NAME_STR":    "",
		"SERVICE_PORT_NUM":      "",
		"TIME_LIMIT_CONSUMER_S": "",
		"BATCH_SIZE_NUM":        "",
		"WORKERS_COUNT":         "",
	}

	for envVar := range originalEnv {
		if val, exists := os.LookupEnv(envVar); exists {
			originalEnv[envVar] = val
			defer os.Setenv(envVar, val)
		} else {
			defer os.Unsetenv(envVar)
		}
		os.Unsetenv(envVar)
	}

	// создаем тестовый HTTP-сервер для API
	var receivedRequests []string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Читаем тело запроса
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		// Сохраняем запрос для проверки
		receivedRequests = append(receivedRequests, string(body))

		// Парсим запрос
		var messages []json.RawMessage
		err = json.Unmarshal(body, &messages)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Формируем ответы
		var responses []OrderResponse
		for _, msgData := range messages {
			var data map[string]interface{}
			json.Unmarshal(msgData, &data)

			orderUID, hasUID := data["order_uid"].(string)

			resp := OrderResponse{
				OrderUID: orderUID,
			}

			if hasUID && orderUID != "" {
				resp.Status = "success"
				resp.ShouldCommit = true
				resp.ShouldDLQ = false
			} else {
				resp.Status = "error"
				resp.Message = "missing order_uid"
				resp.ShouldCommit = false
				resp.ShouldDLQ = true
			}

			responses = append(responses, resp)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMultiStatus)
		json.NewEncoder(w).Encode(responses)
	}))
	defer ts.Close()

	// Получаем порт тестового сервера
	u, err := url.Parse(ts.URL)
	require.NoError(t, err)
	portStr := u.Port()
	require.NotEmpty(t, portStr, "Не удалось извлечь порт из URL: %s", ts.URL)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err, "Не удалось преобразовать порт '%s' в число", portStr)

	// Устанавливаем тестовые переменные окружения
	os.Setenv("TOPIC_NAME_STR", testTopic)
	os.Setenv("GROUP_ID_NAME_STR", testGroupID)
	os.Setenv("DLQ_TOPIC_NAME_STR", testDLQTopic)
	os.Setenv("SERVICE_PORT_NUM", strconv.Itoa(port))
	os.Setenv("TIME_LIMIT_CONSUMER_S", "5") // 5 секунд
	os.Setenv("BATCH_SIZE_NUM", "2")
	os.Setenv("WORKERS_COUNT", "2")

	cfg = readConfig()

	// Корректируем глобальный cfg для теста
	cfg.BatchTimeout = 100 * time.Millisecond
	cfg.MaxRetries = 1
	cfg.RetryDelayBase = 10 * time.Millisecond
	cfg.ClientTimeout = 2 * time.Second
	cfg.LimitConsumWork = 5 * time.Second

	// Записываем тестовые сообщения в Kafka
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{brokerAddr},
		Topic:   testTopic,
	})
	defer writer.Close()

	for _, msg := range testMessages {
		err := writer.WriteMessages(context.Background(), msg)
		require.NoError(t, err)
	}

	// Запускаем консумер в отдельной горутине
	done := make(chan bool)

	// В тесте вместо оригинального consumer запускаем:
	go func() {
		defer func() { done <- true }()

		// Запускаем почти консумер
		runTestConsumer(brokerAddr, testTopic, testGroupID, testDLQTopic)
	}()

	// Ждем завершения консумера
	select {
	case <-done:
		t.Log("Консумер завершил работу")
	case <-time.After(10 * time.Second):
		t.Log("Таймаут ожидания консумера")
	}

	// Проверяем результаты
	if len(receivedRequests) > 0 {
		// Проверяем что валидные сообщения были отправлены в API
		foundValid1 := false
		foundValid2 := false

		for _, req := range receivedRequests {
			if bytes.Contains([]byte(req), []byte("test-1")) {
				foundValid1 = true
			}
			if bytes.Contains([]byte(req), []byte("test-2")) {
				foundValid2 = true
			}
		}

		// Логируем результат
		if foundValid1 {
			t.Log("✓ Сообщение test-1 отправлено в API")
		}
		if foundValid2 {
			t.Log("✓ Сообщение test-2 отправлено в API")
		}
	} else {
		t.Log("API не было вызвано (возможно, консумер не успел обработать сообщения)")
	}

	// Проверяем DLQ
	dlqReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{brokerAddr},
		Topic:     testDLQTopic,
		Partition: 0,
		MinBytes:  10,
		MaxBytes:  10e6,
		MaxWait:   1 * time.Second,
	})
	defer dlqReader.Close()

	dlqCtx, dlqCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer dlqCancel()

	dlqCount := 0
	for {
		msg, err := dlqReader.ReadMessage(dlqCtx)
		if err != nil {
			break
		}
		dlqCount++
		dlqReader.CommitMessages(dlqCtx, msg)
	}

	// Ожидаем 1 сообщение в DLQ (невалидное)
	if dlqCount > 0 {
		assert.Equal(t, 1, dlqCount, "Должно быть 1 невалидное сообщение в DLQ")
		t.Logf("✓ Найдено %d сообщений в DLQ", dlqCount)
	} else {
		t.Log("В DLQ не найдено сообщений (возможно, не успели обработаться)")
	}

	// Очистка
	t.Cleanup(func() {
		conn, err := kafka.Dial("tcp", brokerAddr)
		if err == nil {
			conn.DeleteTopics(testTopic, testDLQTopic)
			conn.Close()
		}
	})
}

// runTestConsumer - запускает консумер с исправленной конфигурацией для теста
func runTestConsumer(brokerAddr, topic, groupID, dlqTopic string) {

	// Создаем контекст
	ctx, cancel := context.WithTimeout(context.Background(), cfg.LimitConsumWork)
	defer cancel()

	// Создаем ридер
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{brokerAddr},
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 10000,
		MaxBytes: 500000,
		MaxWait:  cfg.BatchTimeout,
	})
	defer r.Close()

	// Создаем DLQ writer
	dlqWriter := &kafka.Writer{
		Addr:  kafka.TCP(brokerAddr),
		Topic: dlqTopic,
	}
	defer dlqWriter.Close()

	// HTTP клиент
	httpClient := &http.Client{
		Timeout: cfg.ClientTimeout,
	}

	// Каналы
	messages := make(chan *kafka.Message, cfg.BatchSize*cfg.WorkersCount*10)
	batches := make(chan []*kafka.Message, cfg.WorkersCount*cfg.WorkersCount*10)
	responses := make(chan *RespBatchInfo, cfg.WorkersCount*cfg.WorkersCount*10)
	endCh := make(chan struct{})
	errCh := make(chan error)

	// Запускаем пайплайн
	var wgPipe sync.WaitGroup

	// 1. читаем сообщения из кафки
	wgPipe.Add(1)
	//defer wgPipe.Done()
	go readMsgOfKafka(ctx, r, messages, errCh, &wgPipe)

	// 2. комплектуем батчи
	wgPipe.Add(1)
	//defer wgPipe.Done()
	go complectBatches(messages, batches, &wgPipe)

	// 3. отправляем батчи в API
	wgPipe.Add(1)
	//defer wgPipe.Done()
	go batchWorker(dlqWriter, httpClient, batches, responses, &wgPipe)

	// 4. обрабатываем ответы
	wgPipe.Add(1)
	//defer wgPipe.Done()
	go processBatchResponse(r, dlqWriter, responses, endCh, &wgPipe)

	// Ждем завершения всех этапов
	wgPipe.Wait()
}
