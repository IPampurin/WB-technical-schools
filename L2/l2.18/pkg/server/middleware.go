package server

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

// responseWriter для захвата статуса
type responseWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader для получения статуса в ответе сервера
// (переопределяет метод для сохранения статуса ответа)
func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code                    // сохраняем статус в структуру для захвата статуса
	rw.ResponseWriter.WriteHeader(code) // вызываем оригинальный метод
}

// LoggingMiddleware создает middleware логирования HTTP запросов
// (принимает логгер и возвращает функцию-обертку для обработчиков)
func LoggingMiddleware(logger *log.Logger) func(http.Handler) http.Handler {
	// возвращаем функцию, которая принимает следующий обработчик (next)
	return func(next http.Handler) http.Handler {
		// возвращаем новый обработчик, который логирует и вызывает next
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// фиксируем время получения запроса
			requestTime := time.Now()

			// создаем экземпляр ResponseWriter для захвата статуса
			rw := &responseWriter{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			// вызываем следующий обработчик в цепочке (это может быть другой middleware или финальный хендлер)
			next.ServeHTTP(rw, r)

			// формируем строку лога: время метод путь статус IP
			logLine := requestTime.Format("2006-01-02 15:04:05") + " " + // время получения запроса
				r.Method + " " + // метод запроса
				r.URL.Path + " " + // эндпойнт
				fmt.Sprintf("%d", rw.status) + " " + // статус
				r.RemoteAddr // IP откуда был запрос

			logger.Println(logLine)
		})
	}
}
