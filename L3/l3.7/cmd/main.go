package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/IPampurin/WarehouseControlAntiPattern/pkg/auth"
	"github.com/IPampurin/WarehouseControlAntiPattern/pkg/configuration"
	"github.com/IPampurin/WarehouseControlAntiPattern/pkg/db"
	"github.com/IPampurin/WarehouseControlAntiPattern/pkg/server"
	"github.com/IPampurin/WarehouseControlAntiPattern/pkg/service"
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
		"WarehouseControlAntiPattern",
		"prod",
		logger.WithLevel(logger.InfoLevel),
	)
	if err != nil {
		log.Fatalf("Ошибка создания логгера: %v", err)
	}

	// инициализация JWT секретным ключом
	if cfg.AuthKey.SecretKey == "" {
		appLogger.Warn("ВНИМАНИЕ: SECRET_KEY_JWT не задан, используется значение по умолчанию (небезопасно)")
		auth.Init("default-secret-key") // устанавливаем запасное значение
	} else {
		auth.Init(cfg.AuthKey.SecretKey)
	}

	// получаем экземпляр БД
	storageDB, err := db.InitDB(ctx, &cfg.DB, appLogger)
	if err != nil {
		appLogger.Error("ошибка подключения к БД", "error", err)
		return
	}
	defer func() { _ = db.CloseDB(storageDB) }()

	// получаем экземпляр слоя бизнес-логики
	service := service.InitService(ctx, storageDB)

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
