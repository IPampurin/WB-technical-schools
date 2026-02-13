package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/IPampurin/calendar-server/pkg/storage"
)

// Answer - универсальная структура для возврата ответа
type Answer struct {
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
}

/* POST /create_event
Content-Type: application/json
{
  "user_id": 123,
  "date": "2026-01-15",
  "title": "Встреча",
  "content": "Описание"
}
*/
// CreateEventHandler обрабатывет запрос на добавление события
func (api *API) CreateEventHandler(w http.ResponseWriter, r *http.Request) {

	var answer Answer
	var buf bytes.Buffer

	// req структура для парсинга параметров запроса
	var req struct {
		UserID  int    `json:"user_id"`
		Date    string `json:"date"`
		Title   string `json:"title"`
		Content string `json:"content,omitempty"`
	}

	// читаем запрос
	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		answer.Error = fmt.Sprintf("невозможно прочитать тело запроса %v", err.Error())
		WriterJSON(w, http.StatusBadRequest, answer) // 400
		return
	}

	// определяем структуру
	err = json.Unmarshal(buf.Bytes(), &req)
	if err != nil {
		answer.Error = fmt.Sprintf("невозможно десериализовать тело запроса %v", err.Error())
		WriterJSON(w, http.StatusBadRequest, answer) // 400
		return
	}

	// валидируем входные данные
	// парсим дату
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		answer.Error = fmt.Sprintf("используйте YYYY-MM-DD, неверный формат даты, ошибка: %v", err.Error())
		WriterJSON(w, http.StatusBadRequest, answer) // 400
		return
	}

	// проверяем id
	if req.UserID <= 0 {
		answer.Error = "user_id должен быть положительным числом"
		WriterJSON(w, http.StatusBadRequest, answer) // 400
		return
	}

	// проверяем title
	if req.Title == "" {
		answer.Error = "поле title должно быть заполнено"
		WriterJSON(w, http.StatusBadRequest, answer) // 400
		return
	}

	// вызываем storage
	id, err := api.Storage.Create(req.UserID, date, req.Title, req.Content)
	if err != nil {
		answer.Error = err.Error()
		WriterJSON(w, http.StatusServiceUnavailable, answer) // 503
		return
	}

	answer.Result = fmt.Sprintf("событие создано, ID: %d", id)

	WriterJSON(w, http.StatusCreated, answer) // 201 тут логичнее
}

/*
POST /update_event
{
  "id": 5,
  "user_id": 123,
  "date": "2026-01-15",
  "title": "Новое название",
  "content": "Новое описание"
}
*/
// UpdateEventHandler обрабатывет запрос на обновление события
func (api *API) UpdateEventHandler(w http.ResponseWriter, r *http.Request) {

	var answer Answer
	var buf bytes.Buffer

	// структура для парсинга запроса
	var req struct {
		ID      int    `json:"id"`      // ID события
		UserID  int    `json:"user_id"` // ID пользователя
		Date    string `json:"date"`    // новая дата
		Title   string `json:"title"`   // новый заголовок
		Content string `json:"content,omitempty"`
	}

	// читаем запрос
	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		answer.Error = fmt.Sprintf("невозможно прочитать тело запроса %v", err.Error())
		WriterJSON(w, http.StatusBadRequest, answer) // 400
		return
	}

	// определяем структуру
	err = json.Unmarshal(buf.Bytes(), &req)
	if err != nil {
		answer.Error = fmt.Sprintf("невозможно десериализовать тело запроса %v", err.Error())
		WriterJSON(w, http.StatusBadRequest, answer) // 400
		return
	}

	// парсим дату
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		answer.Error = fmt.Sprintf("используйте YYYY-MM-DD, неверный формат даты, ошибка: %v", err.Error())
		WriterJSON(w, http.StatusBadRequest, answer) // 400
		return
	}

	// проверка обязательных полей
	if req.ID <= 0 {
		answer.Error = "ID события должен быть положительным числом"
		WriterJSON(w, http.StatusBadRequest, answer) // 400
		return
	}
	if req.UserID <= 0 {
		answer.Error = "user_id должен быть положительным числом"
		WriterJSON(w, http.StatusBadRequest, answer) // 400
		return
	}
	if req.Title == "" {
		answer.Error = "title не может быть пустым"
		WriterJSON(w, http.StatusBadRequest, answer) // 400
		return
	}

	// создаем экземпляр события
	event := &storage.Event{
		ID:      req.ID,
		UserID:  req.UserID,
		Date:    date,
		Title:   req.Title,
		Content: req.Content,
	}

	// вызываем storage
	if err := api.Storage.Update(event); err != nil {
		answer.Error = err.Error()
		WriterJSON(w, http.StatusServiceUnavailable, answer) // 503
		return
	}

	answer.Result = "событие обновлено"

	WriterJSON(w, http.StatusOK, answer) // 200
}

/*
POST /delete_event
{
  "user_id": 123,
  "event_id": 5
}
*/
// DeleteEventHandler обрабатывет запрос на удаление события
func (api *API) DeleteEventHandler(w http.ResponseWriter, r *http.Request) {

	var answer Answer
	var buf bytes.Buffer

	// структура для парсинга запроса
	var req struct {
		UserID  int `json:"user_id"`
		EventID int `json:"event_id"`
	}

	// читаем запрос
	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		answer.Error = fmt.Sprintf("невозможно прочитать тело запроса %v", err.Error())
		WriterJSON(w, http.StatusBadRequest, answer) // 400
		return
	}

	// определяем структуру
	err = json.Unmarshal(buf.Bytes(), &req)
	if err != nil {
		answer.Error = fmt.Sprintf("невозможно десериализовать тело запроса %v", err.Error())
		WriterJSON(w, http.StatusBadRequest, answer) // 400
		return
	}

	// проверка обязательных полей
	if req.EventID <= 0 {
		answer.Error = "ID события должен быть положительным числом"
		WriterJSON(w, http.StatusBadRequest, answer) // 400
		return
	}
	if req.UserID <= 0 {
		answer.Error = "user_id должен быть положительным числом"
		WriterJSON(w, http.StatusBadRequest, answer) // 400
		return
	}

	// вызываем storage
	if err := api.Storage.Delete(req.UserID, req.EventID); err != nil {
		answer.Error = err.Error()
		WriterJSON(w, http.StatusServiceUnavailable, answer) // 503
		return
	}

	answer.Result = "событие удалено"

	WriterJSON(w, http.StatusOK, answer) // 200
}

// GET /events_for_day?user_id=123&date=2026-01-15
// GetEventsForDayHandler обрабатывет запрос на чтение событий дня
func (api *API) GetEventsForDayHandler(w http.ResponseWriter, r *http.Request) {

	var answer Answer

	// парсим query параметры
	userIDStr := r.URL.Query().Get("user_id")
	dateStr := r.URL.Query().Get("date")

	// проверяем
	userID, err := strconv.Atoi(userIDStr)
	if err != nil || userID <= 0 {
		answer.Error = "неверный user_id"
		WriterJSON(w, http.StatusBadRequest, answer)
		return
	}
	if dateStr == "" {
		answer.Error = "параметр date обязателен"
		WriterJSON(w, http.StatusBadRequest, answer)
		return
	}

	// парсим дату
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		answer.Error = "неверный формат даты (используйте YYYY-MM-DD)"
		WriterJSON(w, http.StatusBadRequest, answer)
		return
	}

	// вызываем storage
	events, err := api.Storage.GetForDay(userID, date)
	if err != nil {
		answer.Error = err.Error()
		WriterJSON(w, http.StatusServiceUnavailable, answer) // 503
		return
	}

	answer.Result = events

	WriterJSON(w, http.StatusOK, answer) // 200
}

// GET /events_for_week?user_id=123&date=2026-01-15
// GetEventsForWeekHandler обрабатывет запрос на чтение событий недели
func (api *API) GetEventsForWeekHandler(w http.ResponseWriter, r *http.Request) {

	var answer Answer

	// парсим query параметры
	userIDStr := r.URL.Query().Get("user_id")
	dateStr := r.URL.Query().Get("date")

	// проверяем
	userID, err := strconv.Atoi(userIDStr)
	if err != nil || userID <= 0 {
		answer.Error = "неверный user_id"
		WriterJSON(w, http.StatusBadRequest, answer)
		return
	}
	if dateStr == "" {
		answer.Error = "параметр date обязателен"
		WriterJSON(w, http.StatusBadRequest, answer)
		return
	}

	// парсим дату
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		answer.Error = "неверный формат даты (используйте YYYY-MM-DD)"
		WriterJSON(w, http.StatusBadRequest, answer)
		return
	}

	// вызываем storage
	events, err := api.Storage.GetForWeek(userID, date)
	if err != nil {
		answer.Error = err.Error()
		WriterJSON(w, http.StatusServiceUnavailable, answer) // 503
		return
	}

	answer.Result = events

	WriterJSON(w, http.StatusOK, answer) // 200
}

// GET /events_for_month?user_id=123&date=2026-01-15
// GetEventsForMonthHandler обрабатывет запрос на чтение событий месяца
func (api *API) GetEventsForMonthHandler(w http.ResponseWriter, r *http.Request) {

	var answer Answer

	// парсим query параметры
	userIDStr := r.URL.Query().Get("user_id")
	dateStr := r.URL.Query().Get("date")

	// проверяем
	userID, err := strconv.Atoi(userIDStr)
	if err != nil || userID <= 0 {
		answer.Error = "неверный user_id"
		WriterJSON(w, http.StatusBadRequest, answer)
		return
	}
	if dateStr == "" {
		answer.Error = "параметр date обязателен"
		WriterJSON(w, http.StatusBadRequest, answer)
		return
	}

	// парсим дату
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		answer.Error = "неверный формат даты (используйте YYYY-MM-DD)"
		WriterJSON(w, http.StatusBadRequest, answer)
		return
	}

	// вызываем storage
	events, err := api.Storage.GetForMonth(userID, date)
	if err != nil {
		answer.Error = err.Error()
		WriterJSON(w, http.StatusServiceUnavailable, answer) // 503
		return
	}

	answer.Result = events

	WriterJSON(w, http.StatusOK, answer) // 200
}
