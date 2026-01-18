package tests

import (
	"io"
	"log"
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
		_, err := w.Write([]byte("Server is running"))
		if err != nil {
			log.Printf("ошибка при формировании тестового ответа: %v\n", err)
		}
	})

	return r
}

// TestServerStart тестирует запускается ли сервер в принципе
func TestServerStart(t *testing.T) {

	if testing.Short() {
		t.Skip("Пропускаем тест в short режиме.")
	}

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
