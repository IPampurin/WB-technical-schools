package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/handlers"
	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/shutdown"
	"github.com/go-chi/chi/v5"
)

// Run запускает сервер и блокируется до graceful shutdown
func Run(ctx context.Context) error {

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

	// горутина для graceful shutdown
	go func() {
		log.Println("***** горутина, ожидающая сигнала отмены, в server.go запустилась *****")
		// ждём сигнала отмены
		<-ctx.Done()
		log.Println("Получен сигнал завершения, начинаем graceful shutdown...")

		// переключаем флаг
		shutdown.StartShutdown()
		log.Println("Приложение помечено как останавливающееся")

		// останавливаем сервер (до окончания текущего соединения или 30 секунд)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Println("***** srv.Shutdown(shutdownCtx) в server.go пропустил ошибку*****")
			log.Printf("Ошибка при остановке сервера: %v\n", err)
		} else {
			log.Println("Сервер корректно остановлен")
		}
	}()

	// запускаем сервер (блокирующий вызов)
	log.Printf("Запуск сервера на порту %s", port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Println("***** ListenAndServe() в server.go пропустил ошибку*****")
		return fmt.Errorf("ошибка сервера: %w", err)
	}

	return nil
}
