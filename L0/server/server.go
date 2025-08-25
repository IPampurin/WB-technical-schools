package server

import (
	"fmt"
	"net/http"
	"os"

	"github.com/IPampurin/WB-technical-schools/L0/handlers"
	"github.com/go-chi/chi/v5"
)

func Run() error {

	// по умолчанию порт хоста 8081 (доступ в браузере на localhost:8081)
	port, ok := os.LookupEnv("L0_PORT")
	if !ok {
		port = "8081"
	}

	r := chi.NewRouter() // роутер

	// хендлеры
	r.Get("/orders", handlers.GetOrders)
	r.Post("/order", handlers.PostOrder)
	r.Get("/order/{order_uid}", handlers.GetOrderByID)
	r.Delete("/order/{order_uid}", handlers.DeleteOrder)

	// запускаем сервер
	return http.ListenAndServe(fmt.Sprintf(":%v", port), r)
}
