package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServer проверяет корректность работы основных маршрутов сервера
func TestServer(t *testing.T) {
	// Создаем тестовый сервер
	ts := httptest.NewServer(chi.NewRouter())
	defer ts.Close()

	// Получаем базовый URL сервера
	baseURL := ts.URL
	client := &http.Client{}

	t.Run("Test root route", func(t *testing.T) {
		resp, err := client.Get(baseURL)
		require.NoError(t, err, "Ошибка при запросе к корневому маршруту")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Неверный статус-код для корневого маршрута")
	})

	t.Run("Test get orders", func(t *testing.T) {
		resp, err := client.Get(baseURL + "/orders")
		require.NoError(t, err, "Ошибка при запросе к /orders")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Неверный статус-код для /orders")
	})

	t.Run("Test post order", func(t *testing.T) {
		reqBody := `{"order_uid": "test-uid", "purchase_amount": 1000}`
		req, err := http.NewRequest(http.MethodPost, baseURL+"/order", strings.NewReader(reqBody))
		require.NoError(t, err, "Ошибка создания POST-запроса")
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err, "Ошибка при отправке POST-запроса")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode, "Неверный статус-код для POST /order")
	})

	t.Run("Test get order by ID", func(t *testing.T) {
		resp, err := client.Get(baseURL + "/order/test-uid")
		require.NoError(t, err, "Ошибка при запросе к /order/{order_uid}")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Неверный статус-код для /order/{order_uid}")
	})

	t.Run("Test delete order", func(t *testing.T) {
		// Используем http.NewRequest с методом DELETE
		req, err := http.NewRequest(http.MethodDelete, baseURL+"/order/test-uid", nil)
		require.NoError(t, err, "Ошибка создания DELETE-запроса")

		resp, err := client.Do(req)
		require.NoError(t, err, "Ошибка при отправке DELETE-запроса")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode, "Неверный статус-код для DELETE /order/{order_uid}")
	})
}
