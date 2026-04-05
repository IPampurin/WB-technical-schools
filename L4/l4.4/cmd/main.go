package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/IPampurin/GC-MetricsServer/internal/configuration"
	"github.com/IPampurin/GC-MetricsServer/internal/server"
)

func main() {

	// считываем конфигурацию
	cfg, err := configuration.Load()
	if err != nil {
		log.Fatalf("ошибка конфигурации: %v", err)
	}

	// контекст для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// горутина обработки сигналов
	go handleSignals(ctx, cancel)

	// создаём экземпляр сервера
	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("ошибка создания сервера: %v", err)
	}

	// запускаем сервер
	if err := srv.Run(ctx); err != nil {
		log.Printf("сервер завершен с ошибкой: %v", err)
		os.Exit(1)
	}

	log.Println("Сервер успешно остановлен.")
}

// handleSignals слушает SIGINT/SIGTERM и отменяет контекст
func handleSignals(ctx context.Context, cancel context.CancelFunc) {

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	select {
	case <-ctx.Done():
		return
	case <-sigChan:
		cancel()
		return
	}
}
