package tests

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupRouter создаёт тестовый роутер
func setupRouter() http.Handler {
	r := chi.NewRouter()

	// добавляем минимальный маршрут для проверки
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Server is running"))
	})

	return r
}

// TestServerStart тестирует запускается сервер в принципе
func TestServerStart(t *testing.T) {

	// создаем тестовый сервер
	ts := httptest.NewServer(setupRouter())
	defer ts.Close()

	// проверяем доступность сервера
	resp, err := http.Get(ts.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	// проверяем статус ответа
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// дополнительно проверяем содержимое ответа
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "Server is running", string(body))
}
