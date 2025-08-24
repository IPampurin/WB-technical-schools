package main

import (
	"fmt"

	"github.com/IPampurin/WB-technical-schools/L0/db"
	"github.com/IPampurin/WB-technical-schools/L0/server"
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
	err = db.InitDB()
	if err != nil {
		fmt.Printf("ошибка вызова db.Init: %v\n", err)
		return
	}

	// запускаем сервер
	err = server.Run()
	if err != nil {
		fmt.Printf("ошибка запуска сервера: %v\n", err)
		return
	}
}
