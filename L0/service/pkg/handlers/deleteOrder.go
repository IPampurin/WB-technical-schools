package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/db"
	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/models"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

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

	/*
		если данный заказ засветился в кэше, удаляем его из кэша
	*/

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
