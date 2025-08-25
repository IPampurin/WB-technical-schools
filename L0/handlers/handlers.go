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
	/*
		// начинаем транзакцию
		tx := db.DB.Db.Begin()
		defer tx.Rollback()
	*/
	// пишем данные в базу с учётом ассоциаций - заполняем все таблицы согласно структуре models
	resultDB := db.DB.Db.Session(&gorm.Session{FullSaveAssociations: true}).Create(&order)
	if resultDB.Error != nil {
		http.Error(w, resultDB.Error.Error(), http.StatusBadRequest)
		return
	}
	/*
		// коммитим успешную транзакцию
		if err := tx.Commit(); err != nil {
			log.Printf("Ошибка при коммите транзакции: %v", err)
			http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
			return
		}
	*/
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
