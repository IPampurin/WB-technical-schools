package tests

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/IPampurin/calendar-server/pkg/api"
	"github.com/IPampurin/calendar-server/pkg/server"
	"github.com/IPampurin/calendar-server/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServer_InitRoutes проверяет, что маршруты зарегистрированы
// (не запуская реальный сервер)
func TestServer_InitRoutes(t *testing.T) {

	// создаём зависимости
	store := storage.NewStorage()
	api := api.NewAPI(store)

	// инициализируем маршруты без запуска
	mux := http.NewServeMux()
	mux.HandleFunc("POST /create_event", api.CreateEventHandler)
	mux.HandleFunc("POST /update_event", api.UpdateEventHandler)
	mux.HandleFunc("POST /delete_event", api.DeleteEventHandler)
	mux.HandleFunc("GET /events_for_day", api.GetEventsForDayHandler)
	mux.HandleFunc("GET /events_for_week", api.GetEventsForWeekHandler)
	mux.HandleFunc("GET /events_for_month", api.GetEventsForMonthHandler)

	tests := []struct {
		method string
		path   string
	}{
		{"POST", "/create_event"},
		{"POST", "/update_event"},
		{"POST", "/delete_event"},
		{"GET", "/events_for_day"},
		{"GET", "/events_for_week"},
		{"GET", "/events_for_month"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			// 404 - маршрут не зарегистрирован
			// 400/405 - маршрут есть, но запрос кривой
			assert.NotEqual(t, http.StatusNotFound, w.Code, "Маршрут %s %s не зарегистрирован", tt.method, tt.path)
		})
	}
}

// TestServer_Shutdown проверяет graceful shutdown
// (только что не падает, реальный порт не открываем)
func TestServer_Shutdown(t *testing.T) {

	store := storage.NewStorage()
	//apiMock := api.NewAPI(store)

	// создаём сервер с тестовым адресом
	srv := &http.Server{
		Addr:    ":0", // :0 = случайный свободный порт
		Handler: http.DefaultServeMux,
	}

	// инициализируем API (монтируем маршруты)
	api.Init(store)

	// запускаем в горутине
	go func() {
		_ = srv.ListenAndServe()
	}()

	// даём микросекунду на старт
	time.Sleep(10 * time.Millisecond)

	// выключаем
	err := srv.Shutdown(context.Background())
	assert.NoError(t, err)
}

// TestLoggingMiddleware проверяет, что middleware корректно логирует HTTP-запросы
// (создаёт тестовый обработчик, оборачивает его в middleware и проверяет содержимое лога)
func TestLoggingMiddleware(t *testing.T) {

	// перехватываем вывод лога в буфер, чтобы потом проверить, что туда записалось
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	// тестовый обработчик — имитируем реальный хендлер API
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// оборачиваем обработчик в middleware, передаём логгер
	middleware := server.LoggingMiddleware(logger)(handler)

	// создаём тестовый GET-запрос к /test
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// вызываем middleware
	middleware.ServeHTTP(rec, req)

	// проверяем, что в логе есть метод, путь и статус ответа
	logOutput := buf.String()
	assert.Contains(t, logOutput, "GET")
	assert.Contains(t, logOutput, "/test")
	assert.Contains(t, logOutput, "200")
}

// TestSetupLogging проверяет создание логгера и файла для логов
// (убеждается, что функция не возвращает ошибку, а логгер и файл созданы)
func TestSetupLogging(t *testing.T) {

	// вызываем настройку логирования
	logger, file, err := server.SetupLogging()
	require.NoError(t, err, "SetupLogging не должна возвращать ошибку")
	require.NotNil(t, logger, "логгер должен быть создан")
	require.NotNil(t, file, "файл логов должен быть открыт")

	// закрываем файл и удаляем временную папку с логами
	file.Close()
	os.RemoveAll("logs")
}
