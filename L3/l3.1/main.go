package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/IPampurin/DelayedNotifier/pkg/cache"
	"github.com/IPampurin/DelayedNotifier/pkg/configuration"
	"github.com/IPampurin/DelayedNotifier/pkg/consumer"
	"github.com/IPampurin/DelayedNotifier/pkg/db"
	"github.com/IPampurin/DelayedNotifier/pkg/rabbit"
	"github.com/IPampurin/DelayedNotifier/pkg/scheduler"
	"github.com/IPampurin/DelayedNotifier/pkg/server"
	"github.com/wb-go/wbf/logger"
)

func main() {

	// cоздаём контекст
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// запускаем горутину обработки сигналов
	go func() {
		<-sigChan
		cancel()
	}()

	var err error

	// считываем .env файл
	cfg, err := configuration.ReadConfig()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// настраиваем логгер
	appLogger, err := logger.InitLogger(
		logger.ZapEngine,
		"delayed-notifier",
		os.Getenv("APP_ENV"), // пока оставим пустым
		logger.WithLevel(logger.InfoLevel),
	)
	if err != nil {
		log.Fatalf("Ошибка создания логгера: %v", err)
	}
	defer func() { _ = appLogger.(*logger.ZapAdapter) }()

	// подключаем базу данных
	err = db.InitDB(&cfg.DB)
	if err != nil {
		appLogger.Error("ошибка подключения к БД", "error", err)
		return
	}
	defer func() { _ = db.CloseDB() }()

	// инициализируем кэш
	err = cache.InitRedis(&cfg.Redis)
	if err != nil {
		appLogger.Warn("кэш не работает", "error", err)
	}

	// запускаем RabbitMQ
	err = rabbit.InitRabbit(&cfg.RabbitMQ, &cfg.Consumer, appLogger)
	if err != nil {
		appLogger.Error("ошибка подключения к RabbitMQ", "error", err)
		return
	}

	// запускаем горутину-консумер
	go func() {
		err := consumer.InitConsumer(ctx, &cfg.Consumer, appLogger)
		if err != nil {
			appLogger.Error("консумер завершился с ошибкой", "error", err)
			cancel()
		}
	}()

	// запускаем горутину-scheduler
	go func() {
		err := scheduler.InitScheduler(ctx, &cfg.Scheduler, appLogger)
		if err != nil {
			appLogger.Error("планировщик завершился с ошибкой", "error", err)
			cancel()
		}
	}()

	// запускаем сервер
	err = server.Run(ctx, &cfg.Server, appLogger)
	if err != nil {
		appLogger.Error("Ошибка сервера", "error", err)
		cancel()
		return
	}

	appLogger.Info("Приложение корректно завершено")
}
