package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/cache"
	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/db"
	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/models"
	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/shutdown"
	"gorm.io/gorm"
)

// прометеус метрики для сервиса
var (
	// RPS сервиса (обработанные сообщения) - как в продюсере messagesSent
	// -   RPS: rate(service_messages_processed_total[5s])
	// - Total: service_messages_processed_total
	serviceMessagesProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "service_messages_processed_total",
		Help: "Количество обработанных сообщений в сервисе",
	})

	// время работы с БД - аналогично consumerApiResponseTime
	// - Среднее время: avg(rate(service_db_operation_duration_seconds_sum[15s])) / avg(rate(service_db_operation_duration_seconds_count[15s]))
	serviceDBDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "service_db_operation_duration_seconds",
		Help:    "Время выполнения операций с базой данных",
		Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1},
	})
)

// IncomingMessage структура для входящих сообщений от консумера
type IncomingMessage struct {
	Data        json.RawMessage `json:"data"`        // данные заказа
	Traceparent string          `json:"traceparent"` // Traceparent для трейсинга
}

// OrderResponse структура для ответов по каждому сообщению в батче от консумера
type OrderResponse struct {
	OrderUID   string `json:"order_uid"`         // идентификатор сообщения
	Status     string `json:"status"`            // статус: "success", "conflict", "badRequest" (в DLQ), "error" (в DLQ)
	MessageErr string `json:"message,omitempty"` // информация об ошибке
}

// PostOrder принимает json с информацией о заказе и сохраняет данные в базе
func PostOrder(w http.ResponseWriter, r *http.Request) {

	startTime := time.Now()

	// проверяем не останавливается ли сервер
	if shutdown.IsShuttingDown() {
		http.Error(w, "Сервер находится в процессе остановки. Операция невозможна.", http.StatusServiceUnavailable)
		return
	}

	// извлекаем контекст трейсинга из HTTP заголовков (для общего span батча)
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

	// создаем span для обработки всего батча
	ctx, batchSpan := otel.Tracer("order-service").Start(ctx, "service.order.batch",
		trace.WithAttributes(
			attribute.String("component", "order-service"),
			attribute.String("http.method", r.Method),
			attribute.String("http.url", r.URL.Path),
		))
	defer batchSpan.End()

	var buf bytes.Buffer

	// читаем тело запроса
	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		batchSpan.SetStatus(codes.Error, "Ошибка чтения тела запроса")
		return
	}

	// приводим поступившие данные к единому формату
	incomingMessages, err := parseIncomingData(buf.Bytes())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// если нет сообщений
	if len(incomingMessages) == 0 {
		log.Println("Поступил пустой массив заказов в обработчик.")
		http.Error(w, "Пустой массив заказов", http.StatusBadRequest)
		batchSpan.SetStatus(codes.Error, "Пустой массив сообщений")
		return
	}

	// добавляем атрибуты о размере батча
	batchSpan.SetAttributes(
		attribute.Int("batch.size", len(incomingMessages)),
	)

	log.Printf("Получено %d заказов для обработки", len(incomingMessages))

	// парсим данные и извлекаем orderUID для каждого сообщения
	orders := make([]*models.Order, 0, len(incomingMessages))
	orderUIDs := make([]string, 0, len(incomingMessages))
	orderMap := make(map[string]*models.Order) // для быстрого доступа по orderUID
	cacheKeys := make([]string, 0, len(incomingMessages))

	// массив для сохранения spans каждого сообщения
	messageSpans := make([]trace.Span, len(incomingMessages))

	for i, incomingMsg := range incomingMessages {

		var order models.Order

		// парсим данные заказа
		if err := json.Unmarshal(incomingMsg.Data, &order); err != nil {
			log.Printf("Ошибка парсинга данных заказа %d: %v", i, err)
			continue
		}

		// извлекаем traceparent из сообщения и создаем контекст для трейсинга
		var msgCtx context.Context
		if incomingMsg.Traceparent != "" {
			carrier := propagation.HeaderCarrier{}
			carrier.Set("traceparent", incomingMsg.Traceparent)
			msgCtx = otel.GetTextMapPropagator().Extract(ctx, carrier)
		} else {
			msgCtx = ctx
		}

		// создаем span для обработки этого сообщения
		msgCtx, msgSpan := otel.Tracer("order-service").Start(msgCtx, "service.order.process",
			trace.WithAttributes(
				attribute.String("order.uid", order.OrderUID),
				attribute.Int("message.index", i),
			))
		messageSpans[i] = msgSpan

		orders = append(orders, &order)
		orderUIDs = append(orderUIDs, order.OrderUID)
		orderMap[order.OrderUID] = &order
		cacheKeys = append(cacheKeys, fmt.Sprintf("order:%s", order.OrderUID))
	}

	// обеспечиваем завершение всех spans сообщений
	defer func() {
		for i := range messageSpans {
			if messageSpans[i] != nil {
				messageSpans[i].End()
			}
		}
	}()

	// Если не удалось распарсить ни одного заказа
	if len(orders) == 0 {
		http.Error(w, "Не удалось распарсить ни одного заказа", http.StatusBadRequest)
		batchSpan.SetStatus(codes.Error, "Не удалось распарсить ни одного заказа")
		return
	}

	// групповая проверка валидации
	validOrders := make([]*models.Order, 0, len(orders))
	validationResults := make(map[string]error)

	for _, order := range orders {
		if err := validateOrder(order); err != nil {
			validationResults[order.OrderUID] = err
		} else {
			validOrders = append(validOrders, order)
		}
	}

	// групповая проверка в Redis кэше (pipeline)
	cacheDuplicates := make(map[string]bool)
	if len(validOrders) > 0 {
		existsMap, err := cache.BatchGetKeys(cacheKeys)
		if err != nil {
			log.Printf("Ошибка групповой проверки в кэше: %v", err)
			// при ошибке проверяем по одному (fallback)
			for i, key := range cacheKeys {
				if _, err := cache.GetCache(key); err == nil {
					cacheDuplicates[orderUIDs[i]] = true
				}
			}
		} else {
			for i, key := range cacheKeys {
				if existsMap[key] {
					cacheDuplicates[orderUIDs[i]] = true
				}
			}
		}
	}

	// групповая проверка в БД
	dbDuplicates := make(map[string]bool)
	if len(validOrders) > 0 {
		var existingOrders []models.Order
		// один запрос для всех заказов
		if err := db.DB.Db.Where("order_uid IN ?", orderUIDs).Find(&existingOrders).Error; err != nil {
			log.Printf("Ошибка групповой проверки в БД: %v", err)
		} else {
			for _, existing := range existingOrders {
				dbDuplicates[existing.OrderUID] = true
			}
		}
	}

	// 5. подготавливаем ответы и данные для сохранения
	responses := make([]OrderResponse, len(orders))
	ordersToSave := make([]*models.Order, 0, len(orders))

	for i, order := range orders {
		orderUID := order.OrderUID

		// проверяем валидацию
		if err, ok := validationResults[orderUID]; ok {
			responses[i] = OrderResponse{
				OrderUID:   orderUID,
				Status:     "badRequest",
				MessageErr: err.Error(),
			}
			continue
		}

		// проверяем дубликаты в кэше
		if cacheDuplicates[orderUID] {
			responses[i] = OrderResponse{
				OrderUID:   orderUID,
				Status:     "conflict",
				MessageErr: "заказ уже существует в кэше",
			}
			continue
		}

		// проверяем дубликаты в БД
		if dbDuplicates[orderUID] {
			responses[i] = OrderResponse{
				OrderUID:   orderUID,
				Status:     "conflict",
				MessageErr: "заказ уже существует в базе",
			}
			continue
		}

		// заказ готов к сохранению
		ordersToSave = append(ordersToSave, order)
	}

	// групповое сохранение в БД (в транзакции)
	if len(ordersToSave) > 0 {

		saveOrderResults := saveOrdersBatch(ordersToSave)

		// обновляем ответы на основе результатов сохранения
		for i, order := range orders {
			if responses[i].Status != "" {
				continue // уже есть ответ
			}

			if result, ok := saveOrderResults[order.OrderUID]; ok {
				responses[i] = result
			}
		}
	}

	// заполняем ответы для заказов, которые не попали в сохранение
	for i, resp := range responses {
		if resp.Status == "" {
			// если статус пустой, значит что-то пошло не так
			responses[i] = OrderResponse{
				OrderUID:   orders[i].OrderUID,
				Status:     "error",
				MessageErr: "неизвестная ошибка обработки",
			}
		}
	}

	// после обработки всех заказов обновляем метрики
	serviceMessagesProcessed.Add(float64(len(orders)))
	duration := time.Since(startTime).Seconds()
	serviceDBDuration.Observe(duration)

	batchSpan.SetAttributes(
		attribute.Float64("batch.duration_seconds", duration),
		attribute.Int("batch.orders_count", len(orders)),
	)

	// определяем общий HTTP статус
	allSuccess := true
	for _, resp := range responses {
		if resp.Status != "success" && resp.Status != "conflict" {
			allSuccess = false
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if allSuccess {
		w.WriteHeader(http.StatusCreated) // 201
	} else {
		w.WriteHeader(http.StatusMultiStatus) // 207
	}

	// возвращаем массив ответов
	if err := json.NewEncoder(w).Encode(responses); err != nil {
		log.Printf("Ошибка кодирования ответа: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	successCount := countByStatus(responses, "success")
	conflictCount := countByStatus(responses, "conflict")
	errorCount := len(responses) - successCount - conflictCount

	log.Printf("Обработка завершена. Успешно: %d, Дубликатов: %d, Ошибок: %d",
		successCount, conflictCount, errorCount)
}

// parseIncomingData помогает распарсить входящие данные, так как от консумера
// приходят структуры с трейсом, а из веб-интерфейса - json
func parseIncomingData(data []byte) ([]IncomingMessage, error) {

	// пробуем парсить как массив IncomingMessage (формат от консумера)
	var messages []IncomingMessage
	if err := json.Unmarshal(data, &messages); err == nil {
		// проверяем, что есть поле Data
		if len(messages) > 0 && len(messages[0].Data) != 0 {
			return messages, nil
		}
	}

	// если не получилось, пробуем парсить как массив Order (формат от веб-интерфейса)
	var orders []models.Order
	if err := json.Unmarshal(data, &orders); err != nil {
		return nil, fmt.Errorf("не удалось распарсить данные: %v", err)
	}

	// конвертируем в IncomingMessage
	result := make([]IncomingMessage, len(orders))
	for i, order := range orders {
		orderJSON, err := json.Marshal(order)
		if err != nil {
			return nil, fmt.Errorf("ошибка маршалинга заказа: %v", err)
		}
		result[i] = IncomingMessage{
			Data:        orderJSON,
			Traceparent: "", // нет traceparent от веб-интерфейса
		}
	}

	return result, nil
}

// saveOrdersBatch сохраняет заказы пачкой в транзакции
func saveOrdersBatch(orders []*models.Order) map[string]OrderResponse {

	results := make(map[string]OrderResponse)

	if len(orders) == 0 {
		return results
	}

	log.Printf("Начинаем транзакцию для сохранения %d заказов.", len(orders))
	tx := db.DB.Db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Printf("Паника при сохранении заказов: %v", r)
			// помечаем все заказы как ошибки
			for _, order := range orders {
				results[order.OrderUID] = OrderResponse{
					OrderUID:   order.OrderUID,
					Status:     "error",
					MessageErr: "внутренняя ошибка сервера",
				}
			}
		}
	}()

	// создаем сессию с нужными настройками
	session := tx.Session(&gorm.Session{
		FullSaveAssociations: true,
		CreateBatchSize:      1000, // вставка в БД батчами по 1000
	})

	// сохраняем все заказы разом
	result := session.Create(orders)
	if result.Error != nil {
		tx.Rollback()
		log.Printf("Ошибка при сохранении заказов: %v", result.Error)
		for _, order := range orders {
			results[order.OrderUID] = OrderResponse{
				OrderUID:   order.OrderUID,
				Status:     "error",
				MessageErr: result.Error.Error(),
			}
		}
		return results
	}

	// проверяем коммит
	if commitResult := tx.Commit(); commitResult.Error != nil {
		log.Printf("Ошибка при коммите транзакции: %v", commitResult.Error)
		for _, order := range orders {
			results[order.OrderUID] = OrderResponse{
				OrderUID:   order.OrderUID,
				Status:     "error",
				MessageErr: "ошибка сохранения в базу",
			}
		}
		return results
	}

	log.Println("Транзакция успешно завершена.")

	// групповое кэширование (одним pipeline)
	if len(orders) > 0 {
		keyValues := make(map[string]interface{})
		for _, order := range orders {
			key := fmt.Sprintf("order:%s", order.OrderUID)
			keyValues[key] = order
		}

		if err := cache.BatchSet(keyValues); err != nil {
			log.Printf("Ошибка группового кэширования: %v", err)
			// fallback: сохраняем по одному
			for _, order := range orders {
				cacheKey := fmt.Sprintf("order:%s", order.OrderUID)
				if err := cache.SetCache(cacheKey, order); err != nil {
					log.Printf("Ошибка кэширования заказа %s: %v", order.OrderUID, err)
				}
			}
		} else {
			log.Printf("Успешно закэшировано %d заказов", len(orders))
		}
	}

	// добавляем успешные ответы
	for _, order := range orders {
		results[order.OrderUID] = OrderResponse{
			OrderUID:   order.OrderUID,
			Status:     "success",
			MessageErr: "заказ успешно добавлен в базу",
		}
	}

	return results
}

// countByStatus подсчитывает ответы по статусу
func countByStatus(responses []OrderResponse, status string) int {

	count := 0
	for _, resp := range responses {
		if resp.Status == status {
			count++
		}
	}
	return count
}

// validate указатель на валидируемую структуру
var validate = validator.New()

// validateOrder проверяет заполненность полей поступивших данных
func validateOrder(order *models.Order) error {

	err := validate.Struct(order)
	if err != nil {
		// преобразуем ошибки валидации в читаемый формат
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			var errorMessages []string
			for _, fieldError := range validationErrors {
				errorMessages = append(errorMessages,
					fmt.Sprintf("Поле %s: %s", fieldError.Field(), fieldError.Tag()))
			}
			return fmt.Errorf("ошибки валидации: %v", errorMessages)
		}
	}

	return err
}
