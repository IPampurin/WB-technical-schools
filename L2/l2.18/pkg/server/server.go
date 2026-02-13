package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/IPampurin/calendar-server/pkg/api"
	"github.com/IPampurin/calendar-server/pkg/storage"
)

const calendarPortDefault = "8081"

// Run запускает сервер и передаёт объект хранилища хэндлерам
func Run(db storage.Repository) error {

	port, ok := os.LookupEnv("CALENDAR_PORT")
	if !ok {
		port = calendarPortDefault
	}

	// инициализируем api
	api.Init(db)

	// настраиваем логирование
	logger, logFile, err := SetupLogging()
	if err != nil {
		return fmt.Errorf("ошибка настройки логирования: %w", err)
	}
	defer logFile.Close()

	// оборачиваем в middleware
	handler := LoggingMiddleware(logger)(http.DefaultServeMux)

	// создаем и настраиваем сервер
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: handler,
	}

	// канал для shutdown
	idleConnsClosed := make(chan struct{})

	// http.Handle("/", http.FileServer(http.Dir("web"))) // здесь может быть запуск фронтэнда

	// горутина для graceful shutdown
	go func() {

		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)
		<-sigint

		logger.Println("Получен сигнал остановки сервера.")

		// останавливаем сервер (до окончания текущего соединения или 30 секунд)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("Ошибка при остановке сервера: %v\n", err)
		}

		close(idleConnsClosed)
	}()

	// запускаем сервер
	logger.Printf("Сервер запущен на %s\n", srv.Addr)

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("ошибка сервера: %w", err)
	}

	// ждём остановки сервера
	<-idleConnsClosed
	logger.Println("Сервер корректно остановлен.")

	return nil
}
