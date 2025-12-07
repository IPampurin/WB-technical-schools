package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-playground/validator/v10"

	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/cache"
	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/db"
	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/models"
	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/shutdown"
	"gorm.io/gorm"
)

// OrderResponse структура для ответов по каждому сообщению в батче от консумера
type OrderResponse struct {
	OrderUID     string `json:"orderUID"`          // идентификатор сообщения
	Status       string `json:"status"`            // статус: "success", "conflict", "badRequest", "error"
	Message      string `json:"message,omitempty"` // информация об ошибке
	ShouldCommit bool   `json:"shouldCommit"`      // можно ли коммитить в кафке
	ShouldDLQ    bool   `json:"shouldDLQ"`         // надо ли отправить в DLQ
}

// PostOrder принимает json с информацией о заказе и сохраняет данные в базе
func PostOrder(w http.ResponseWriter, r *http.Request) {

	// проверяем не останавливается ли сервер
	if shutdown.IsShuttingDown() {
		http.Error(w, "Сервер находится в процессе остановки. Операция невозможна.", http.StatusServiceUnavailable)
		return
	}

	var orders []*models.Order
	var buf bytes.Buffer

	// читаем тело запроса
	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// парсим json из запроса в структуру заказа
	if err = json.Unmarshal(buf.Bytes(), &orders); err != nil {
		http.Error(w, "Ожидается JSON массив заказов", http.StatusBadRequest)
		return
	}

	// если нет заказов
	if len(orders) == 0 {
		http.Error(w, "Пустой массив заказов", http.StatusBadRequest)
		return
	}

	log.Printf("Получено %d заказов для обработки", len(orders))

	// подготавливаем данные для групповых проверок
	orderUIDs := make([]string, len(orders))
	orderMap := make(map[string]*models.Order) // для быстрого доступа по orderUID
	cacheKeys := make([]string, len(orders))

	for i, order := range orders {
		orderUIDs[i] = order.OrderUID
		orderMap[order.OrderUID] = order
		cacheKeys[i] = fmt.Sprintf("order:%s", order.OrderUID)
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
		if err := db.DB.Db.Where("orderUID IN ?", orderUIDs).Find(&existingOrders).Error; err != nil {
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
				OrderUID:     orderUID,
				Status:       "badRequest",
				Message:      err.Error(),
				ShouldCommit: false,
				ShouldDLQ:    true,
			}
			continue
		}

		// проверяем дубликаты в кэше
		if cacheDuplicates[orderUID] {
			responses[i] = OrderResponse{
				OrderUID:     orderUID,
				Status:       "conflict",
				Message:      "заказ уже существует в кэше",
				ShouldCommit: true,
				ShouldDLQ:    false,
			}
			continue
		}

		// проверяем дубликаты в БД
		if dbDuplicates[orderUID] {
			responses[i] = OrderResponse{
				OrderUID:     orderUID,
				Status:       "conflict",
				Message:      "заказ уже существует в базе",
				ShouldCommit: true,
				ShouldDLQ:    false,
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
				OrderUID:     orders[i].OrderUID,
				Status:       "error",
				Message:      "неизвестная ошибка обработки",
				ShouldCommit: false,
				ShouldDLQ:    true,
			}
		}
	}

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
					OrderUID:     order.OrderUID,
					Status:       "error",
					Message:      "внутренняя ошибка сервера",
					ShouldCommit: false,
					ShouldDLQ:    true,
				}
			}
		}
	}()

	// создаем сессию с нужными настройками
	session := tx.Session(&gorm.Session{
		FullSaveAssociations: true,
	})

	// сохраняем все заказы разом
	result := session.Create(orders)
	if result.Error != nil {
		tx.Rollback()
		log.Printf("Ошибка при сохранении заказов: %v", result.Error)
		for _, order := range orders {
			results[order.OrderUID] = OrderResponse{
				OrderUID:     order.OrderUID,
				Status:       "error",
				Message:      result.Error.Error(),
				ShouldCommit: false,
				ShouldDLQ:    true,
			}
		}
		return results
	}

	// проверяем коммит
	if commitResult := tx.Commit(); commitResult.Error != nil {
		log.Printf("Ошибка при коммите транзакции: %v", commitResult.Error)
		for _, order := range orders {
			results[order.OrderUID] = OrderResponse{
				OrderUID:     order.OrderUID,
				Status:       "error",
				Message:      "ошибка сохранения в базу",
				ShouldCommit: false,
				ShouldDLQ:    true,
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
			OrderUID:     order.OrderUID,
			Status:       "success",
			Message:      "заказ успешно добавлен в базу",
			ShouldCommit: true,
			ShouldDLQ:    false,
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
