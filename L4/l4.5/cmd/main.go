package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "net/http/pprof" // неявно регистрирует /debug/pprof/ на http.DefaultServeMux

	"github.com/IPampurin/pprofSimpleService/internal/configuration"
	"github.com/IPampurin/pprofSimpleService/internal/server"
	"github.com/IPampurin/pprofSimpleService/internal/service"
)

func main() {

	// считываем конфигурацию
	cfg, err := configuration.Load()
	if err != nil {
		log.Fatalf("ошибка конфигурации: %v", err)
	}

	// заводим конрневой контекст
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// запускаем ожидание сигналов
	go signalHandler(ctx, cancel)

	// получаем экземпляр логики
	svc := service.NewService()

	// получаем сервер
	srv := server.NewServer(&cfg, svc)

	// запускаем pprof-сервер в отдельной горутине
	go func() {
		log.Println("pprof-сервер запущен на порту :6060")
		if err := http.ListenAndServe(":6060", nil); err != nil {
			log.Printf("ошибка pprof-сервера: %v", err)
		}
	}()

	// запускаем сервер
	if err := srv.Run(ctx); err != nil {
		log.Printf("сервер остановлен с ошибкой: %v", err)
	}

	log.Println("Приложение корректно остановлено.")
}

// signalHandler слушает SIGINT/SIGTERM и отменяет контекст
func signalHandler(ctx context.Context, cancel context.CancelFunc) {

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
