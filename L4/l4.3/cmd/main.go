package main

import (
	"log"
	"os"

	"github.com/IPampurin/EventCalendar/internal/app"
	"github.com/IPampurin/EventCalendar/internal/configuration"
)

func main() {

	cfg, err := configuration.Load()
	if err != nil {
		log.Fatalf("ошибка конфигурации: %v", err)
	}

	application, err := app.New(&cfg)
	if err != nil {
		log.Fatalf("ошибка инициализации приложения: %v", err)
	}
	defer application.Close()

	if err := application.Run(); err != nil {
		log.Printf("приложение завершено с ошибкой: %v", err)
		os.Exit(1)
	}
}
