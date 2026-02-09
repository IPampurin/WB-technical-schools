package main

import (
	"fmt"

	"github.com/IPampurin/calendar-server/pkg/server"
	"github.com/IPampurin/calendar-server/pkg/storage"
)

func main() {

	// создаём хранилище
	db := storage.NewStorage()

	// запускаем сервер
	if err := server.Run(db); err != nil {
		fmt.Printf("Ошибка запуска сервера: %v\n", err)
		return
	}
}
