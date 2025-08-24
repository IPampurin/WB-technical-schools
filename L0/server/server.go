package server

import (
	"fmt"
	"net/http"
	"os"

	"github.com/IPampurin/WB-technical-schools/L0/db"
	"github.com/IPampurin/WB-technical-schools/L0/handlers"
	"github.com/go-chi/chi/v5"
)

func Run() error {

	db.ConnectDB()

	// по умолчанию порт хоста 8081 (доступ в браузере на localhost:8081)
	port, ok := os.LookupEnv("L0_PORT")
	if !ok {
		port = "8081"
	}

	r := chi.NewRouter()

	r.Get("/order", handlers.GetTasks)
	r.Post("/order", handlers.PostTask)
	r.Get("/order/{id}", handlers.GetTaskByID)
	r.Delete("/order/{id}", handlers.DeleteTask)

	return http.ListenAndServe(fmt.Sprintf(":%v", port), r)
}
