package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"mywget/pkg/loader"
)

func main() {

	// проверяем аргументы запуска программы
	if len(os.Args) < 2 {
		fmt.Println("Использование: mywget <URL> [глубина]")
		os.Exit(1)
	}

	// контекст для отмены работы программы
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// заводим и регистрируем канал для обработки Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)

	// горутина для обработки Ctrl+C
	go func() {
		<-sigChan
		fmt.Println("Получен сигнал остановки, завершаем работу...")
		cancel()
	}()

	url := os.Args[1]
	depth := 1
	if len(os.Args) >= 3 {
		d, _ := strconv.Atoi(os.Args[2])
		depth = d
	}

	err := loader.Load(ctx, url, depth)
	if err != nil {
		fmt.Printf("Загрузка %s завершилась с ошибкой: %v\n", url, err)
		os.Exit(1)
	} else {
		fmt.Printf("Загрузка %s завершилась успешно.\n", url)
		os.Exit(0)
	}
}
