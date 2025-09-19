/* ВАРИАНТ №5 - решение с приёмом параметра и graceful shutdown по нажатию Ctrl + C (запуск: go run 5main.go --workers 'X') */

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// worker печатает в консоль значение, полученное из канала
func worker(ctx context.Context, ch chan int, wg *sync.WaitGroup) {

	defer wg.Done() // уменьшаем счётчик WaitGroup

	for {
		// постоянно слушаем два канала
		select {
		case v, ok := <-ch: // и если основной канал не закрыт, печатаем то, что пришло
			if !ok {
				return // если канал закрыт, завершаем работу
			}
			fmt.Println(v)
		case <-ctx.Done():
			return // при получении сигнала по каналу отмены завершаем worker
		}
	}
}

func main() {

	// определяем количество запускаемых при запуске программы воркеров
	// по умолчанию воркеров будет 4
	countWorkers := flag.Int("workers", 4, "count of workers")
	flag.Parse()

	if *countWorkers <= 0 {
		log.Fatal("Количество воркеров должно быть больше ноля.")
	}

	// для корректного завершения работы воркеров определяем контекст
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// организуем WaitGroup
	var wg sync.WaitGroup

	in := make(chan int, *countWorkers) // канал для передачи данных

	// запускаем заданное количество воркеров
	for i := 0; i < *countWorkers; i++ {
		wg.Add(1) // увеличиваем счётчик WaitGroup
		go worker(ctx, in, &wg)
	}

	// sigChan канал для получения сигналов ОС, единичный буфер гарантирует вычитываение
	sigChan := make(chan os.Signal, 1)
	// регистрируем канал для получения сигналов прерывания программы пользователем (Ctrl+C)
	// или мягкого завершения окружением
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	number := 0 // число для отправки в канал передачи данных

loop:
	for {
		select {
		// если можем, отправляем число в канал in
		case in <- number:
			number++
			// если же есть сигнал прерывания, выходим из цикла
		case <-sigChan:
			break loop
		}
	}

	cancel()  // сигнал воркерам закругляться
	close(in) // закрываем in
	wg.Wait() // ждём остановки всех воркеров
}
