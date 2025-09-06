package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/db"
	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/models"
)

// GetOrders выводит список всех заказов с учётом параметров пагинации и общим количеством
func GetOrders(w http.ResponseWriter, r *http.Request) {

	// инициализируем переменные для пагинации
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	// значения по умолчанию
	page := 1
	limit := 10

	// если в параметрах запроса нет лимитов вывода данных,
	// установим значения по умолчанию
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// получаем общее количество заказов
	var total int64
	if err := db.DB.Db.Model(&models.Order{}).Count(&total).Error; err != nil {
		log.Printf("Ошибка при получении общего количества заказов: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// получаем заказы с учётом пагинации
	var orders []models.Order

	query := db.DB.Db.Preload("Delivery").Preload("Payment").Preload("Items").Offset((page - 1) * limit).Limit(limit)

	if err := query.Find(&orders).Error; err != nil {
		log.Printf("Ошибка при получении заказов: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// формируем ответ
	response := struct {
		Total  int64          `json:"Всего заказов"`
		Page   int            `json:"Страниц для показа"`
		Limit  int            `json:"Показывать на странице по"`
		Orders []models.Order `json:"Данные заказов"`
	}{
		Total:  total,
		Page:   page,
		Limit:  limit,
		Orders: orders,
	}

	// Сериализация ответа
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "    ") // добавляем отступы для читаемости
	if err := encoder.Encode(response); err != nil {
		log.Printf("Ошибка при формировании ответа: %v", err)
	}

	log.Printf("Успешно возвращено %d/%d заказов", len(orders), total)
}
