package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/IPampurin/UrlShortener/pkg/cache"
	"github.com/IPampurin/UrlShortener/pkg/configuration"
	"github.com/IPampurin/UrlShortener/pkg/db"
	"github.com/IPampurin/UrlShortener/pkg/server"
	"github.com/IPampurin/UrlShortener/pkg/service"
	"github.com/wb-go/wbf/logger"
)

func main() {

	// cоздаём контекст
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// запускаем горутину обработки сигналов
	go signalHandler(ctx, cancel)

	// считываем .env файл
	cfg, err := configuration.ReadConfig()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// настраиваем логгер
	appLogger, err := logger.InitLogger(
		logger.ZapEngine,
		"UrlShortener",
		os.Getenv("APP_ENV"), // пока оставим пустым
		logger.WithLevel(logger.InfoLevel),
	)
	if err != nil {
		log.Fatalf("Ошибка создания логгера: %v", err)
	}
	defer func() { _ = appLogger.(*logger.ZapAdapter) }()

	// получаем экземпляр хранилища
	storage, err := db.InitDB(ctx, &cfg.DB, appLogger)
	if err != nil {
		appLogger.Error("ошибка подключения к БД", "error", err)
		return
	}
	defer func() { _ = db.CloseDB(storage) }()

	// получаем экземпляр кэша
	cache, err := cache.InitCache(ctx, storage, &cfg.Redis, appLogger)
	if err != nil {
		appLogger.Warn("кэш не работает", "error", err)
	}

	// получаем экземпляр слоя бизнес-логики
	service := service.InitService(ctx, storage, cache)

	// запускаем сервер
	err = server.Run(ctx, &cfg.Server, service, appLogger)
	if err != nil {
		appLogger.Error("Ошибка сервера", "error", err)
		cancel()
		return
	}

	appLogger.Info("Приложение корректно завершено")
}

// signalHandler обрабатывет сигналы отмены
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
