package handlers

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/IPampurin/WB-technical-schools/L0/db"
	"github.com/IPampurin/WB-technical-schools/L0/models"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func GetOrders(w http.ResponseWriter, r *http.Request) {
	orders := []models.Order{}

	resultDB := db.DB.Db.Find(&orders)
	if resultDB.Error != nil {
		http.Error(w, resultDB.Error.Error(), http.StatusBadRequest)
		return
	}

	resp, err := json.Marshal(orders)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
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
}

func GetOrderByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	order := new(models.Order)

	resultDB := db.DB.Db.First(&order, id)
	if resultDB.Error != nil {
		http.Error(w, resultDB.Error.Error(), http.StatusBadRequest)
		return
	}

	resp, err := json.Marshal(order)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func DeleteOrder(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resultDB := db.DB.Db.Delete(&models.Order{}, id)
	if resultDB.Error != nil {
		http.Error(w, resultDB.Error.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}
