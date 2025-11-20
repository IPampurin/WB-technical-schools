package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/cache"
	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/db"
	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/server"
	"github.com/joho/godotenv"
)

// Глобальная переменная - неизбежное зло для межпакетного доступа
var isShuttingDown int32

// IsShuttingDown экспортируемая функция для проверки состояния
func IsShuttingDown() bool {
	return atomic.LoadInt32(&isShuttingDown) == 1
}

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

	// создаем контекст для graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// запускаем сервер
	var srv *http.Server
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		var err error
		srv, err = server.Run(ctx)
		if err != nil && err != http.ErrServerClosed {
			log.Printf("ошибка запуска сервера: %v\n", err)
			stop() // отправляем сигнал завершения
		}
	}()

	// ожидаем сигнал завершения
	<-ctx.Done()
	log.Println("Получен сигнал завершения, начинаем graceful shutdown...")

	// определяем таймаут в зависимости от нагрузки
	shutdownTimeout := getShutdownTimeout()
	log.Printf("Таймаут graceful shutdown: %v", shutdownTimeout)

	// Останавливаем сервер
	if srv != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(srv); err != nil {
			log.Printf("Ошибка при остановке сервера: %v\n", err)
		} else {
			log.Println("Сервер корректно остановлен")
		}
	}

	// ждем завершения сервера
	wg.Wait()

	log.Println("Приложение корректно завершено")
}

// getShutdownTimeout определяет таймаут в зависимости от времени суток и нагрузки
func getShutdownTimeout() time.Duration {
	// можно добавить иную логику определения таймаута
	// например, ночью - меньше, днем - больше
	hour := time.Now().Hour()

	switch {
	case hour >= 0 && hour < 6: // ночью
		return 15 * time.Second
	case hour >= 18: // вечером
		return 25 * time.Second
	default: // днем
		return 30 * time.Second
	}
}
