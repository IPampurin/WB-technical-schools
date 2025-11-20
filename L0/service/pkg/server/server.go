package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/handlers"
	"github.com/go-chi/chi/v5"
)

var isShuttingDown int32 // atomic flag для graceful shutdown

// Run запускает сервер с поддержкой graceful shutdown
func Run(ctx context.Context) (*http.Server, error) {

	// по умолчанию порт хоста 8081 (доступ в браузере на localhost:8081)
	port, ok := os.LookupEnv("L0_PORT")
	if !ok {
		port = "8081"
	}

	r := chi.NewRouter() // роутер

	// основной контент (фронт)
	mainFiles := http.FileServer(http.Dir("web"))
	r.Handle("/", mainFiles)

	// роуты
	r.Get("/orders", handlers.GetOrders)
	r.Post("/order", handlers.PostOrder)
	r.Get("/order/{order_uid}", handlers.GetOrderByID)
	r.Delete("/order/{order_uid}", handlers.DeleteOrder)

	// создаем экземпляр сервера
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%v", port),
		Handler: r,
	}

	// запускаем сервер в отдельной горутине
	go func() {
		log.Printf("Запуск сервера на порту %s", port)
		if err := srv.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			log.Printf("Ошибка сервера: %v\n", err)
		}
	}()

	return srv, nil
}

// gracefulShutdownMiddleware блокирует новые запросы при shutdown
func gracefulShutdownMiddleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(&isShuttingDown) == 1 {
			http.Error(w, "Сервер находится в процессе остановки. Попробуйте позже.", http.StatusServiceUnavailable)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// StartShutdown помечает сервер как останавливающийся
func StartShutdown() {

	atomic.StoreInt32(&isShuttingDown, 1)
	log.Println("Сервер больше не принимает новые соединения")
}

// Shutdown выполняет graceful shutdown сервера
func Shutdown(srv *http.Server) error {

	if IsShuttingDown() {

	}

	StartShutdown()

	// Даем время на завершение текущих запросов (но не более 30 сек)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return srv.Shutdown(shutdownCtx)
}
