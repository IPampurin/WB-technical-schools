package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/go-playground/validator/v10"

	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/cache"
	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/db"
	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/models"
	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/shutdown"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// PostOrder принимает json с информацией о заказе и сохраняет данные в базе
func PostOrder(w http.ResponseWriter, r *http.Request) {

	// проверяем не останавливается ли сервер
	if shutdown.IsShuttingDown() {
		http.Error(w, "Сервер находится в процессе остановки. Операция невозможна.", http.StatusServiceUnavailable)
		return
	}

	var order *models.Order
	var buf bytes.Buffer

	// читаем тело запроса
	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// парсим json из запроса в структуру заказа
	if err = json.Unmarshal(buf.Bytes(), &order); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// валидируем поступившие данные с помощью пакета validator
	if err = validateOrder(order); err != nil {
		log.Printf("Получены некорректные данные: %v", err)
		http.Error(w, "некорректные данные - не будем сохранять", http.StatusBadRequest)
		return
	}

	// если поступившие данные корректны, проверяем наличие в кэше заказа с поступившим order_uid
	cacheKey := fmt.Sprintf("order:%s", order.OrderUID)
	if _, err := cache.GetCache(cacheKey); err == nil {
		log.Printf("Попытка добавить дубликат заказа (найден в кэше): OrderUID=%s", order.OrderUID)
		http.Error(w, "заказ с таким OrderUID уже существует", http.StatusConflict)
		return
	} else if !errors.Is(err, redis.Nil) { // игнорируем ошибку "ключ не найден"
		log.Printf("Заказ с OrderUID=%s не найден в кэше. Проверяем в базе.", order.OrderUID)
	}

	// проверяем, существует ли уже заказ с таким OrderUID в базе
	var existingOrder models.Order

	if err := db.DB.Db.Where("order_uid = ?", order.OrderUID).First(&existingOrder).Error; err == nil {
		log.Printf("Попытка добавить дубликат заказа: OrderUID=%s", order.OrderUID)
		http.Error(w, "заказ с таким OrderUID уже существует", http.StatusConflict)
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// обрабатываем ошибки, кроме "не найдено"
		log.Printf("Ошибка проверки дубликата: %v", err)
		http.Error(w, "внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	log.Println("Получен заказ:", order)

	log.Println("Начинаем транзакцию.")
	// начинаем транзакцию
	tx := db.DB.Db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		}
	}()

	// создаем сессию с нужными настройками
	session := tx.Session(&gorm.Session{
		FullSaveAssociations: true,
	})

	// сохраняем данные через транзакцию
	result := session.Create(order)
	if result.Error != nil {
		tx.Rollback()
		http.Error(w, result.Error.Error(), http.StatusBadRequest)
		return
	}

	// проверяем коммит
	if commitResult := tx.Commit(); commitResult.Error != nil {
		log.Printf("Ошибка при коммите транзакции: %v", commitResult.Error)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// сохраняем данные в кэше с учётом TTL
	ttl := cache.GetTTL()
	if err := cache.SetCahe(cacheKey, order, ttl); err != nil {
		log.Printf("Ошибка кэширования заказа %s: %v", order.OrderUID, err)
	}

	log.Println("Транзакция успешно завершена.")

	// завершаем работу функции
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	log.Printf("Заказ UID = %s успешно добавлен в базу.", order.OrderUID)
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
