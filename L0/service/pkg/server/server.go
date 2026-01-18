package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/IPampurin/Orders-Info-Menedger/service/pkg/handlers"
	"github.com/IPampurin/Orders-Info-Menedger/service/pkg/shutdown"
	"github.com/go-chi/chi/v5"
)

const servicePortConst = "8081"

// SrvConfig описывает настройки с учётом переменных окружения
type SrvConfig struct {
	ServicePort string // порт, на котором работает сервер
}

var cfgSrv *SrvConfig

// getEnvString проверяет наличие и корректность переменной окружения (строковое значение)
func getEnvString(envVariable, defaultValue string) string {

	value, ok := os.LookupEnv(envVariable)
	if ok {
		return value
	}

	return defaultValue
}

// readConfig уточняет конфигурацию с учётом переменных окружения
func readConfig() *SrvConfig {

	return &SrvConfig{
		ServicePort: getEnvString("SERVICE_PORT", servicePortConst),
	}
}

// Run запускает сервер и блокируется до graceful shutdown
func Run(ctx context.Context) error {

	// считываем конфигурацию
	cfgSrv = readConfig()

	r := chi.NewRouter() // роутер

	// основной контент (фронт)
	mainFiles := http.FileServer(http.Dir("web"))
	r.Handle("/", mainFiles)

	// роуты
	r.Get("/orders", handlers.GetOrders)
	r.Post("/order", handlers.PostOrder)
	r.Get("/order/{order_uid}", handlers.GetOrderByID)
	r.Delete("/order/{order_uid}", handlers.DeleteOrder)

	// создаем экземпляр сервера
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%v", cfgSrv.ServicePort),
		Handler: r,
	}

	// горутина для graceful shutdown
	go func() {
		// ждём сигнала отмены
		<-ctx.Done()
		log.Println("Получен сигнал завершения, начинаем graceful shutdown...")

		// переключаем флаг
		shutdown.StartShutdown()
		log.Println("Приложение помечено как останавливающееся.")

		// останавливаем сервер (до окончания текущего соединения или 30 секунд)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("Ошибка при остановке сервера: %v\n", err)
		} else {
			log.Println("Сервер корректно остановлен")
		}
	}()

	// запускаем сервер (блокирующий вызов)
	log.Printf("Запуск сервера на порту %s", cfgSrv.ServicePort)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("ошибка сервера: %w", err)
	}

	return nil
}
