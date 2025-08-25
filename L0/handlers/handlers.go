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

func PostOrder(w http.ResponseWriter, r *http.Request) {
	orders := new(models.Order)
	var buf bytes.Buffer

	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err = json.Unmarshal(buf.Bytes(), &orders); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Println(orders)

	resultDB := db.DB.Db.Create(&orders)
	if resultDB.Error != nil {
		http.Error(w, resultDB.Error.Error(), http.StatusBadRequest)
		return
	}

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
