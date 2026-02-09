package server

import (
	"fmt"
	"net/http"
	"os"

	"github.com/IPampurin/calendar-server/pkg/api"
	"github.com/IPampurin/calendar-server/pkg/storage"
)

const calendarPortDefault = "8081"

func Run(db storage.Repository) error {

	port, ok := os.LookupEnv("CALENDAR_PORT")
	if !ok {
		port = calendarPortDefault
	}

	// инициализируем api
	api.Init(db)

	// http.Handle("/", http.FileServer(http.Dir("web"))) // здесь может быть запуск фронтэнда

	return http.ListenAndServe(fmt.Sprintf(":%v", port), nil)
}
