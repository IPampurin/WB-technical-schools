package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/IPampurin/calendar-server/pkg/api"
	"github.com/IPampurin/calendar-server/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newMockStorage возвращает новое тестовое хранилище
// (возьмём существующий тип, чтобы методы не переписывать)
func newMockStorage() *storage.Storage {
	return &storage.Storage{
		Events: make(map[int][]*storage.Event),
		NextID: 1,
	}
}

// TestNewAPI проверяет создание экземпляра API
func TestNewAPI(t *testing.T) {

	// создаём тестовое хранилище
	mock := newMockStorage()

	// вызываем функцию создания объекта для api
	api := api.NewAPI(mock)

	// проверяем, что api не пуст
	require.NotNil(t, api, "NewAPI() вернул nil")

	// проверка соответствия интерфейсу (рефлексия)
	assert.Implements(t, (*storage.Repository)(nil), api.Storage, "поле api не реализует интерфейс Repository")
}

// TestCalendarAPI_Integration проверяет полный сценарий работы с календарём
// тест последовательно выполняет операции CRUD через HTTP-эндпоинты и проверяет:
// 1) Корректность HTTP-ответов (статус-коды, структура JSON)
// 2) Фактическое состояние хранилища после каждой операции
func TestCalendarAPI_Integration(t *testing.T) {

	// 1. Инициализируем
	mock := newMockStorage()
	apiMock := api.NewAPI(mock)

	// регистрируем хендлеры в тестовом HTTP сервере
	mux := http.NewServeMux()
	mux.HandleFunc("POST /create_event", apiMock.CreateEventHandler)
	mux.HandleFunc("POST /update_event", apiMock.UpdateEventHandler)
	mux.HandleFunc("POST /delete_event", apiMock.DeleteEventHandler)
	mux.HandleFunc("GET /events_for_day", apiMock.GetEventsForDayHandler)
	mux.HandleFunc("GET /events_for_week", apiMock.GetEventsForWeekHandler)
	mux.HandleFunc("GET /events_for_month", apiMock.GetEventsForMonthHandler)

	// тестовые сервер и клиент
	server := httptest.NewServer(mux)
	defer server.Close()

	client := server.Client()

	// 2. Создаём событие
	t.Run("CREATE event", func(t *testing.T) {
		body := `{
            "user_id": 123,
            "date": "2026-01-15",
            "title": "Встреча",
            "content": "Описание"
        }`

		resp, err := client.Post(server.URL+"/create_event", "application/json", bytes.NewBufferString(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		// проверяем ответ
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var answer api.Answer
		err = json.NewDecoder(resp.Body).Decode(&answer)
		require.NoError(t, err)
		assert.Contains(t, answer.Result, "событие создано, ID: 1")
		assert.Empty(t, answer.Error)

		// проверяем хранилище
		events, err := mock.GetForDay(123, time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))
		require.NoError(t, err)
		assert.Len(t, events, 1)
		assert.Equal(t, "Встреча", events[0].Title)
	})

	// 3. Получаем события дня
	t.Run("GET events for day", func(t *testing.T) {
		url := fmt.Sprintf("%s/events_for_day?user_id=123&date=2026-01-15", server.URL)
		resp, err := client.Get(url)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var answer api.Answer
		err = json.NewDecoder(resp.Body).Decode(&answer)
		require.NoError(t, err)
		assert.Empty(t, answer.Error)

		// проверяем, что вернулось одно событие
		eventsData, err := json.Marshal(answer.Result)
		require.NoError(t, err)

		var events []*storage.Event
		err = json.Unmarshal(eventsData, &events)
		require.NoError(t, err)
		assert.Len(t, events, 1)
		assert.Equal(t, "Встреча", events[0].Title)
	})

	// 4. Обновляем событие
	t.Run("UPDATE event", func(t *testing.T) {
		body := `{
            "id": 1,
            "user_id": 123,
            "date": "2026-01-15",
            "title": "Новое название",
            "content": "Новое описание"
        }`

		resp, err := client.Post(server.URL+"/update_event", "application/json", bytes.NewBufferString(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var answer api.Answer
		err = json.NewDecoder(resp.Body).Decode(&answer)
		require.NoError(t, err)
		assert.Equal(t, "событие обновлено", answer.Result)
		assert.Empty(t, answer.Error)

		// проверяем, что данные обновились в хранилище
		events, err := mock.GetForDay(123, time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))
		require.NoError(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, "Новое название", events[0].Title)
		assert.Equal(t, "Новое описание", events[0].Content)
	})

	// 5. Проверяем неделю
	t.Run("GET events for week", func(t *testing.T) {

		// создаём ещё одно событие через 2 дня
		_, _ = mock.Create(123, time.Date(2026, 1, 17, 0, 0, 0, 0, time.UTC), "Ещё событие", "")

		url := fmt.Sprintf("%s/events_for_week?user_id=123&date=2026-01-15", server.URL)
		resp, err := client.Get(url)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var answer api.Answer
		err = json.NewDecoder(resp.Body).Decode(&answer)
		require.NoError(t, err)

		eventsData, _ := json.Marshal(answer.Result)
		var events []*storage.Event
		_ = json.Unmarshal(eventsData, &events)

		assert.Len(t, events, 2) // два события на неделе
	})

	// 6. Удаляем событие
	t.Run("DELETE event", func(t *testing.T) {
		body := `{
            "user_id": 123,
            "event_id": 1
        }`

		resp, err := client.Post(server.URL+"/delete_event", "application/json", bytes.NewBufferString(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var answer api.Answer
		err = json.NewDecoder(resp.Body).Decode(&answer)
		require.NoError(t, err)
		assert.Equal(t, "событие удалено", answer.Result)

		// проверяем, что событие удалилось
		events, err := mock.GetForDay(123, time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))
		require.NoError(t, err)
		assert.Len(t, events, 0) // события нет
	})

	// 7. Проверяем месяц
	t.Run("GET events for month", func(t *testing.T) {
		url := fmt.Sprintf("%s/events_for_month?user_id=123&date=2026-01-15", server.URL)
		resp, err := client.Get(url)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var answer api.Answer
		err = json.NewDecoder(resp.Body).Decode(&answer)
		require.NoError(t, err)

		eventsData, _ := json.Marshal(answer.Result)
		var events []*storage.Event
		_ = json.Unmarshal(eventsData, &events)

		assert.Len(t, events, 1) // только событие от 17.01
	})

	// 8. Негативные сценарии
	t.Run("NEGATIVE: create with empty title", func(t *testing.T) {
		body := `{"user_id":123,"date":"2026-01-15","title":""}`
		resp, err := client.Post(server.URL+"/create_event", "application/json", bytes.NewBufferString(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("NEGATIVE: update non-existent event", func(t *testing.T) {
		body := `{"id":999,"user_id":123,"date":"2026-01-15","title":"Test"}`
		resp, err := client.Post(server.URL+"/update_event", "application/json", bytes.NewBufferString(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	})
}

// TestWriterJSON проверяет корректность сериализации и отправки JSON-ответа
func TestWriterJSON(t *testing.T) {

	rec := httptest.NewRecorder()
	data := map[string]string{"result": "ok"}
	api.WriterJSON(rec, http.StatusOK, data)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp["result"])
}

// TestWriterJSONError проверяет обработку ошибок при Marshal
func TestWriterJSONError(t *testing.T) {

	rec := httptest.NewRecorder()
	// передаём канал - json.Marshal гарантированно вернёт ошибку
	api.WriterJSON(rec, http.StatusOK, make(chan int))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "fatal error")
}
