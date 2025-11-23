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
)

func main() {

	var err error

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
	log.Println("***** дошли до запуска сервера в main() *****")
	if err := server.Run(ctx); err != nil {
		log.Println("сервер в main() выдал ошибку")
		log.Printf("Ошибка сервера: %v\n", err)
		return
	}

	log.Println("Приложение корректно завершено")
}
