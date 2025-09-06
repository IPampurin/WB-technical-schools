package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/db"
	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/models"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

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

	/*
	   проверяем есть ли такой заказ в кэше и если нет, то
	*/
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

	/*
		а если заказ был кэше, то дальше оперируем слайсом байт из кэша:
		размаршалливаем его в order (?) и

	*/

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
