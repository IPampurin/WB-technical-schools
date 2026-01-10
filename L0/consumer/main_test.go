package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"sync"
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

// TestReadConfig тестирует получение конфигурации
func TestReadConfig(t *testing.T) {
	// Сохраняем оригинальные значения всех переменных окружения
	envVars := map[string]string{
		"TOPIC_NAME_STR":      "",
		"GROUP_ID_NAME_STR":   "",
		"KAFKA_PORT_NUM":      "",
		"SERVICE_PORT":        "",
		"BATCH_SIZE_NUM":      "",
		"BATCH_TIMEOUT_S":     "",
		"MAX_RETRIES_NUM":     "",
		"RETRY_DELEY_BASE_MS": "",
		"COUNT_CLIENT":        "",
		"CLIENT_TIMEOUT_S":    "",
		"DLQ_TOPIC_NAME_STR":  "",
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
				Topic:          topicNameConst,
				GroupID:        groupIDNameConst,
				KafkaHost:      kafkaHostConst,
				KafkaPort:      kafkaPortConst,
				ServiceHost:    serviceHostConst,
				ServicePort:    servicePortConst,
				BatchSize:      batchSizeConst,
				BatchTimeout:   time.Duration(batchTimeoutConst) * time.Second,
				MaxRetries:     maxRetriesConst,
				RetryDelayBase: time.Duration(retryDelayBaseConst) * time.Millisecond,
				CountClient:    countClientConst,
				ClientTimeout:  time.Duration(clientTimeoutConst) * time.Second,
				DlqTopic:       dlqTopicConst,
			},
		},
		{
			name: "частично переопределенные значения",
			setEnv: map[string]string{
				"TOPIC_NAME_STR":    "custom-topic",
				"GROUP_ID_NAME_STR": "custom-group",
				"SERVICE_PORT":      "9090",
				"BATCH_SIZE_NUM":    "500",
				"COUNT_CLIENT":      "20",
			},
			expected: &ConsumerConfig{
				Topic:          "custom-topic",
				GroupID:        "custom-group",
				KafkaHost:      kafkaHostConst,
				KafkaPort:      kafkaPortConst,
				ServiceHost:    serviceHostConst,
				ServicePort:    9090,
				BatchSize:      500,
				BatchTimeout:   time.Duration(batchTimeoutConst) * time.Second,
				MaxRetries:     maxRetriesConst,
				RetryDelayBase: time.Duration(retryDelayBaseConst) * time.Millisecond,
				CountClient:    20,
				ClientTimeout:  time.Duration(clientTimeoutConst) * time.Second,
				DlqTopic:       dlqTopicConst,
			},
		},
		{
			name: "все значения переопределены",
			setEnv: map[string]string{
				"TOPIC_NAME_STR":      "test-topic",
				"GROUP_ID_NAME_STR":   "test-group",
				"KAFKA_PORT_NUM":      "9093",
				"SERVICE_PORT":        "8082",
				"BATCH_SIZE_NUM":      "150",
				"BATCH_TIMEOUT_S":     "10",
				"MAX_RETRIES_NUM":     "5",
				"RETRY_DELEY_BASE_MS": "200",
				"COUNT_CLIENT":        "15",
				"CLIENT_TIMEOUT_S":    "30",
				"DLQ_TOPIC_NAME_STR":  "test-dlq",
			},
			expected: &ConsumerConfig{
				Topic:          "test-topic",
				GroupID:        "test-group",
				KafkaHost:      kafkaHostConst,
				KafkaPort:      9093,
				ServiceHost:    serviceHostConst,
				ServicePort:    8082,
				BatchSize:      150,
				BatchTimeout:   10 * time.Second,
				MaxRetries:     5,
				RetryDelayBase: 200 * time.Millisecond,
				CountClient:    15,
				ClientTimeout:  30 * time.Second,
				DlqTopic:       "test-dlq",
			},
		},
		{
			name: "некорректные числовые значения",
			setEnv: map[string]string{
				"KAFKA_PORT_NUM": "not-a-number",
				"SERVICE_PORT":   "invalid",
				"BATCH_SIZE_NUM": "abc",
				"COUNT_CLIENT":   "xyz",
			},
			expected: &ConsumerConfig{
				Topic:          topicNameConst,
				GroupID:        groupIDNameConst,
				KafkaHost:      kafkaHostConst,
				KafkaPort:      kafkaPortConst,
				ServiceHost:    serviceHostConst,
				ServicePort:    servicePortConst,
				BatchSize:      batchSizeConst,
				BatchTimeout:   time.Duration(batchTimeoutConst) * time.Second,
				MaxRetries:     maxRetriesConst,
				RetryDelayBase: time.Duration(retryDelayBaseConst) * time.Millisecond,
				CountClient:    countClientConst,
				ClientTimeout:  time.Duration(clientTimeoutConst) * time.Second,
				DlqTopic:       dlqTopicConst,
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
			assert.Equal(t, tt.expected.KafkaHost, config.KafkaHost, "KafkaHost не совпадает")
			assert.Equal(t, tt.expected.KafkaPort, config.KafkaPort, "KafkaPort не совпадает")
			assert.Equal(t, tt.expected.ServiceHost, config.ServiceHost, "ServiceHost не совпадает")
			assert.Equal(t, tt.expected.ServicePort, config.ServicePort, "ServicePort не совпадает")
			assert.Equal(t, tt.expected.BatchSize, config.BatchSize, "BatchSize не совпадает")
			assert.Equal(t, tt.expected.BatchTimeout, config.BatchTimeout, "BatchTimeout не совпадает")
			assert.Equal(t, tt.expected.MaxRetries, config.MaxRetries, "MaxRetries не совпадает")
			assert.Equal(t, tt.expected.RetryDelayBase, config.RetryDelayBase, "RetryDelayBase не совпадает")
			assert.Equal(t, tt.expected.CountClient, config.CountClient, "CountClient не совпадает")
			assert.Equal(t, tt.expected.ClientTimeout, config.ClientTimeout, "ClientTimeout не совпадает")
			assert.Equal(t, tt.expected.DlqTopic, config.DlqTopic, "DlqTopic не совпадает")
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

func TestConsumPipelineIntegrations(t *testing.T) {

	if testing.Short() {
		t.Skip("Пропускаем интеграционный тест в short режиме")
	}

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

	require.NoError(t, err, "Не удалось запустить Kafka контейнер")

	// 2. Получаем адрес брокера
	brokers, err := kafkaContainer.Brokers(ctx)
	require.NoError(t, err, "Не удалось получить адрес брокера")
	require.NotEmpty(t, brokers, "Список брокеров пуст")

	brokerAddr := brokers[0]
	t.Logf("Kafka запущена на: %s", brokerAddr)

	// задаём имена топиков
	testTopic := "test-topic-consumer"
	dlqTopic := "test-dlq-consumer"

	// устанавливаем соединение с брокером и создаём топики
	connTestTopic, err := kafka.DialLeader(context.Background(), "tcp", brokerAddr, testTopic, 0)
	if err != nil {
		t.Logf("ошибка создания тестового топика кафки в тесте: %v.\n", err)
	}
	defer func() {
		// закрываем соединение testProduceConn
		if err := connTestTopic.Close(); err != nil {
			t.Logf("ошибка при закрытии соединения c тестовым топиком в тесте: %v.\n", err)
		}
	}()
	connDLQTopic, err := kafka.DialLeader(context.Background(), "tcp", brokerAddr, dlqTopic, 0)
	if err != nil {
		t.Logf("ошибка создания тестового топика DLQ кафки в тесте: %v.\n", err)
	}
	defer func() {
		// закрываем соединение testProduceConn
		if err := connDLQTopic.Close(); err != nil {
			t.Logf("ошибка при закрытии соединения с тестовым топиком DLQ в тесте: %v.\n", err)
		}
	}()

	// 4. Создаём тестовые сообщения (30 штук), каждое пятое без OrderUID (для DLQ)
	testMessages := generateTestMessages(30)

	// 5. Отправляем сообщения в тестовый топик
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokerAddr),
		Topic:        testTopic,
		RequiredAcks: kafka.RequireAll,
	}
	defer func() {
		if err := writer.Close(); err != nil {
			t.Logf("ошибка при закрытии врайтера в тестовый топик Kafka в тесте: %v", err)
		}
	}()

	err = writer.WriteMessages(ctx, testMessages...)
	require.NoError(t, err, "Не удалось записать в кафку тестовые сообщения")

	t.Logf("Отправлено %d тестовых сообщений", len(testMessages))

	// на этом ^ имитация продюсера завершена

	// 6. Создаем тестовый API сервер
	var apiRequests []string
	var apiRequestsMu sync.Mutex
	successCount := 0 // отправленные в API сообщения (с order_uid)

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiRequestsMu.Lock()
		defer apiRequestsMu.Unlock()

		// считываем сообщения из запроса и парсим
		body, _ := io.ReadAll(r.Body)
		apiRequests = append(apiRequests, string(body))

		var messages []json.RawMessage
		if err := json.Unmarshal(body, &messages); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// формируем ответы
		responses := make([]OrderResponse, len(messages))
		for i, msgData := range messages {
			var data map[string]interface{}
			json.Unmarshal(msgData, &data)

			orderUID, _ := data["order_uid"].(string)
			status := "success"
			if orderUID == "" {
				status = "error"
			} else {
				successCount++
			}

			responses[i] = OrderResponse{
				OrderUID: orderUID,
				Status:   status,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMultiStatus)
		json.NewEncoder(w).Encode(responses)
	}))
	defer apiServer.Close()

	// на этом ^ имитация API сервиса завершена

	// 7. Настраиваем конфиг для консумера
	host, portStr, err := net.SplitHostPort(brokerAddr)
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)

	apiPort, _ := strconv.Atoi(apiServer.URL[17:])

	// сохраняем оригинальный конфиг
	origCfg := cfg
	defer func() { cfg = origCfg }()

	// меняем значения в конфиге на более соответствующие тестам
	cfg = &ConsumerConfig{
		Topic:          testTopic,
		GroupID:        "test-group",
		KafkaHost:      host,
		KafkaPort:      port,
		ServiceHost:    "127.0.0.1",
		ServicePort:    apiPort,
		BatchSize:      5,
		BatchTimeout:   100 * time.Millisecond,
		MaxRetries:     1,
		RetryDelayBase: 50 * time.Millisecond,
		CountClient:    2,
		ClientTimeout:  2 * time.Second,
		DlqTopic:       dlqTopic,
	}

	// 8. Инициализируем noop tracer
	tracer = noop.NewTracerProvider().Tracer("test")

	// 9. Создаём врайтер для записи в DLQ
	dlqWriter := &kafka.Writer{
		Addr:  kafka.TCP(brokerAddr),
		Topic: cfg.DlqTopic,
	}
	defer func() {
		if err := dlqWriter.Close(); err != nil {
			t.Logf("ошибка при закрытии dlqWriter в тесте: %v", err)
		}
	}()

	// 10. Создаём ридер из тестового топика
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{brokerAddr},
		Topic:    cfg.Topic,
		GroupID:  cfg.GroupID,
		MinBytes: 10,
		MaxBytes: 10e6,
	})
	defer func() {
		if err := r.Close(); err != nil {
			t.Logf("ошибка при закрытии ридера Kafka в тесте: %v", err)
		}
	}()

	// 11. Организуем каналы
	messagesCh := make(chan *MessageWithTrace, cfg.BatchSize*10) // канал для входящих сообщений с большим буфером
	batchesCh := make(chan []*MessageWithTrace, cfg.BatchSize/4) // канал для передачи батчей на обработку
	preparesCh := make(chan *PrepareBatch, cfg.BatchSize/4)      // канал для передачи подготовленной информации к отправке в api
	collectCh := make(chan []*PrepareBatch, cfg.BatchSize/4)     // канал скомпанованных данных о батчах для параллельной передачи в api
	responsesCh := make(chan *RespBatchInfo, cfg.BatchSize/4)    // канал передачи ответов по батчам и мап с сообщениями

	// канал для передачи ошибки ридера при чтении сообщений из кафки
	errCh := make(chan error)
	// канал для передачи сигнала об окончании обработки сообщений
	endCh := make(chan struct{})

	time.Sleep(3 * time.Second) // тут надо просто дождаться записи сообщений в тестовый топик

	// 12. Запускаем конвейер (собственно, сам этот ужасный тест)

	// чтение прекращается через пару секунд - для тестового количества сообщений достаточно
	ctxPipe, pipeCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer pipeCancel()

	// wgPipe для ожидания всех горутин конвейера
	var wgPipe sync.WaitGroup

	// 1. читаем сообщения из кафки
	wgPipe.Add(1)
	go readMsgOfKafka(ctxPipe, r, messagesCh, errCh, &wgPipe)

	// 2. комплектуем батчи из прочитанных сообщений
	wgPipe.Add(1)
	go complectBatches(messagesCh, batchesCh, &wgPipe)

	// 3. подготавливаем информацию для отправки
	wgPipe.Add(1)
	go prepareBatchToSending(dlqWriter, batchesCh, preparesCh, &wgPipe)

	// 4. собираем данные по батчам в группы для параллельной отправки в api
	wgPipe.Add(1)
	go batchPrepareCollect(preparesCh, collectCh, &wgPipe)

	// 5. параллельно направляем запросы в api по группе батчей
	wgPipe.Add(1)
	go sendBatchInfo(dlqWriter, collectCh, responsesCh, &wgPipe)

	// 6. обрабатываем ответы api для каждого сообщения - коммитим или заполняем DLQ
	wgPipe.Add(1)
	go processBatchResponse(r, dlqWriter, responsesCh, endCh, &wgPipe)

	// close(errCh) уже выполнился при выходе из readMsgOfKafka
	// close(endCh) уже выполнился при выходе из processBatchResponse

	// 13. Ждём завершения работы конвейера
	go func() {
		wgPipe.Wait()
	}()

	// ждём получения ошибки или nil из логики конвейера
	// ошибки: нет возможности читать сообщения из брокера
	err = <-errCh
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		t.Logf("конвейер завершился с критической ошибкой в тесте: %v", err)
		pipeCancel()
	}

	<-endCh // завершился последний воркер в конвейере обработки

	if err == nil {
		t.Log("Конвейер корректно завершил работу в тесте.")
	}

	// 14. Проверяем результаты
	apiRequestsMu.Lock()
	t.Logf("API получило %d запросов", len(apiRequests))
	t.Logf("Успешно обработано сообщений: %d", successCount)
	apiRequestsMu.Unlock()

	// проверяем сообщения в DLQ
	dlqReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{brokerAddr},
		Topic:    cfg.DlqTopic,
		GroupID:  cfg.GroupID + "dlq",
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	defer func() {
		if err := dlqReader.Close(); err != nil {
			t.Logf("ошибка при закрытии dlqReader в тесте: %v\n", err)
		}
	}()

	var dlqMessages []kafka.Message
	dlqCtx, dlqCancel := context.WithTimeout(ctx, 2*time.Second)
	defer dlqCancel()

	for {
		msg, err := dlqReader.FetchMessage(dlqCtx)
		if err != nil {
			break
		}
		dlqMessages = append(dlqMessages, msg)
		dlqReader.CommitMessages(ctx, msg)
	}

	dlqCount := len(dlqMessages)
	t.Logf("В DLQ находится %d сообщений", dlqCount)

	// проверяем, что невалидные сообщения попали в DLQ
	expectedDLQCount := 0
	for i := 0; i < len(testMessages); i++ {
		if (i+1)%5 == 0 {
			expectedDLQCount++
		}
	}

	assert.Equal(t, expectedDLQCount, dlqCount, "Количество сообщений в DLQ должно совпадать с ожидаемым (каждое 5-е)")

	// Проверяем, что API получило только валидные сообщения
	expectedAPICount := len(testMessages) - expectedDLQCount // 30 - 30/5 = 24 сообщения
	assert.Equal(t, expectedAPICount, successCount, "API должно было получить только валидные сообщения")

	t.Log("Тест конвейера потребителя завершен успешно")
}

// generateTestMessages создает тестовые сообщения для Kafka
func generateTestMessages(total int) []kafka.Message {

	messages := make([]kafka.Message, total)

	for i := 0; i < total; i++ {

		var message []byte

		if (i+1)%5 == 0 {
			// каждое 5-е сообщение будет без order_uid (для тестирования DLQ)
			message = []byte(fmt.Sprintf(`{"order_uid": "", "data": "message-%d", "error": "no order uid"}`, i+1))
		} else {
			message = []byte(fmt.Sprintf(`{"order_uid": "test-%d", "data": "message-%d", "valid": true}`, i+1, i+1))
		}

		messages[i] = kafka.Message{
			Key:   []byte(fmt.Sprintf("key-%d", i+1)),
			Value: message,
		}
	}

	return messages
}
