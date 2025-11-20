package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/cache"
	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/db"
	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/server"
	"github.com/joho/godotenv"
)

func main() {

	var err error

	// загружаем переменные окружения
	err = godotenv.Load()
	if err != nil {
		fmt.Printf("ошибка загрузки .env файла: %v\n", err)
	}

	// подключаем базу данных
	err = db.ConnectDB()
	if err != nil {
		fmt.Printf("ошибка вызова db.ConnectDB: %v\n", err)
		return
	}
	defer db.CloseDB()

	// инициализируем кэш
	err = cache.InitRedis()
	if err != nil {
		fmt.Printf("кэш отвалился, ошибка вызова cache.InitRedis: %v\n", err)
	}

	// создаем контекст для сигналов отмены
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// запускаем сервер и ждем его завершения
	if err := server.Run(ctx); err != nil {
		log.Printf("Ошибка сервера: %v\n", err)
		return
	}

	log.Println("Приложение корректно завершено")
}
