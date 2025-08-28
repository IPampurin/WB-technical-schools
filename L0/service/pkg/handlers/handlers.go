package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/db"
	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/models"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// GetOrders выводит список всех заказов с учётом параметров пагинации
func GetOrders(w http.ResponseWriter, r *http.Request) {

	// инициализируем переменные для пагинации
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")

	// если в параметрах запроса нет лимитов вывода данных,
	// установим значения по умолчанию
	pageSize := 10
	if page != "" {
		pageSize, _ = strconv.Atoi(page)
	}
	limitSize := 50
	if limit != "" {
		limitSize, _ = strconv.Atoi(limit)
	}

	var orders []models.Order // слайс для хранения заказов

	// создаём запрос к базе данных с учётом связанных данных
	query := db.DB.Db.Preload("Delivery").Preload("Payment").Preload("Items").Find(&orders)

	// применим пагинацию
	query = query.Offset((pageSize - 1) * limitSize).Limit(limitSize)

	// выполнение запроса
	if query.Error != nil {
		log.Printf("Ошибка при получении заказов: %v", query.Error)
		http.Error(w, query.Error.Error(), http.StatusBadRequest)
		return
	}

	// маршалим даные в JSON с отступами для читаемости
	resp, err := json.MarshalIndent(orders, "", "    ")
	if err != nil {
		log.Printf("Ошибка при маршалинге JSON: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)

	log.Printf("Успешно получено %d заказов", len(orders))
}

// PostOrder принимает json с информацией о заказе и сохраняет данные в базе
func PostOrder(w http.ResponseWriter, r *http.Request) {

	order := new(models.Order)
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

	// валидируем поступившие данные
	if ok, message := validateOrder(order); !ok {
		log.Printf("Получены некорректные данные: %v", message)
		http.Error(w, "некорректные данные - не будем сохранять", http.StatusBadRequest)
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

	log.Println("Транзакция успешно завершена.")

	// завершаем работу функции
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	log.Printf("Заказ UID = %s успешно добавлен в базу.", order.OrderUID)
}

// GetOrderByID выдаёт данные о заказе по order_uid
func GetOrderByID(w http.ResponseWriter, r *http.Request) {

	// получаем OrderUID из параметров запроса
	orderUID := chi.URLParam(r, "order_uid")

	if orderUID == "" {
		log.Printf("Ошибка: order_uid не указан")
		http.Error(w, "Параметр order_uid обязателен", http.StatusBadRequest)
		return
	}

	// создаем экземпляр заказа
	var order models.Order

	// получаем заказ из базы данных
	result := db.DB.Db.First(&order, "order_uid = ?", orderUID)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			log.Printf("Заказ с UID %s не найден", orderUID)
			http.Error(w, "Заказ не найден", http.StatusNotFound)
			return
		}
		log.Printf("Ошибка при получении заказа: %v", result.Error)
		http.Error(w, "Ошибка при получении заказа", http.StatusInternalServerError)
		return
	}

	// маршалим даные в JSON с отступами для читаемости
	resp, err := json.MarshalIndent(order, "", "    ")
	if err != nil {
		log.Printf("Ошибка при маршалинге данных: %v", err)
		http.Error(w, "Ошибка при формировании ответа", http.StatusInternalServerError)
		return
	}

	// формируем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)

	log.Printf("Заказ с UID %s успешно получен", orderUID)
}

// DeleteOrder удаляет заказ и связанные с ним данные в случае
// наличия order_uid в таблице и в параметрах запроса
func DeleteOrder(w http.ResponseWriter, r *http.Request) {

	// получаем OrderUID из параметров запроса
	orderUID := chi.URLParam(r, "order_uid")

	if orderUID == "" {
		log.Printf("Ошибка: order_uid не указан")
		http.Error(w, "Параметр order_uid обязателен", http.StatusBadRequest)
		return
	}

	log.Println("Начинаем транзакцию.")
	// начинаем транзакцию
	tx := db.DB.Db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		}
	}()

	// создаем сессию
	session := tx.Session(&gorm.Session{})

	// проверяем существование заказа перед удалением
	var order models.Order
	result := session.First(&order, "order_uid = ?", orderUID)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			log.Printf("Заказ с UID %s не найден", orderUID)
			http.Error(w, "Заказ не найден", http.StatusNotFound)
			return
		}
		tx.Rollback()
		log.Printf("Ошибка при проверке заказа: %v", result.Error)
		http.Error(w, "Ошибка при проверке заказа", http.StatusInternalServerError)
		return
	}

	// удаляем связанные данные через сессию
	if err := session.Model(&order).Association("Items").Clear(); err != nil {
		tx.Rollback()
		log.Printf("Ошибка при очистке связанных данных: %v", err)
		http.Error(w, "Ошибка при удалении связанных данных", http.StatusInternalServerError)
		return
	}

	// удаляем сам заказ
	result = session.Delete(&order)
	if result.Error != nil {
		tx.Rollback()
		log.Printf("Ошибка при удалении заказа: %v", result.Error)
		http.Error(w, "Ошибка при удалении заказа", http.StatusInternalServerError)
		return
	}

	// проверяем коммит
	if commitResult := tx.Commit(); commitResult.Error != nil {
		log.Printf("Ошибка при коммите транзакции: %v", commitResult.Error)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	log.Println("Транзакция успешно завершена.")
	log.Printf("Заказ с UID %s успешно удален", orderUID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}

// validateOrder проверяет заполненность полей поступивших данных
// и комментирует некорректность для последующего логирования
func validateOrder(order *models.Order) (bool, string) {

	// валидация Order
	if order.OrderUID == "" ||
		order.TrackNumber == "" ||
		order.CustomerID == "" ||
		order.Locale == "" ||
		order.DeliveryService == "" ||
		order.Shardkey == "" ||
		order.SMID == 0 {
		return false, "Поля order должны быть заполнены."
	}

	// валидация Delivery
	if order.Delivery.Name == "" ||
		order.Delivery.Phone == "" ||
		order.Delivery.Zip == "" ||
		order.Delivery.City == "" ||
		order.Delivery.Address == "" ||
		order.Delivery.Region == "" ||
		order.Delivery.Email == "" {
		return false, "Поля delivery должны быть заполнены."
	}

	// валидация Payment
	if order.Payment.Transaction == "" ||
		order.Payment.Currency == "" ||
		order.Payment.Provider == "" ||
		order.Payment.Amount <= 0 ||
		order.Payment.PaymentDT <= 0 ||
		order.Payment.Bank == "" ||
		order.Payment.DeliveryCost < 0 ||
		order.Payment.GoodsTotal < 0 ||
		order.Payment.CustomFee < 0 {
		return false, "Поля payment должны быть заполнены, да ещё и корректно."
	}

	// валидация Items
	if len(order.Items) == 0 {
		return false, "Товар должен быть хотя бы один - items не корректен."
	}
	for _, item := range order.Items {
		if item.ChrtID == 0 ||
			item.Price <= 0 ||
			item.RID == "" ||
			item.Name == "" ||
			item.TotalPrice <= 0 ||
			item.NMID == 0 ||
			item.Status == 0 {
			return false, "Поля items должны быть заполнены, да ещё и корректно."
		}
	}

	return true, ""
}
