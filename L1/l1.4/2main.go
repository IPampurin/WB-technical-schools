/* ВАРИАНТ №2 - решение с применением context и остановкой программы по Ctrl + C */

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const n = 5 // количество горутин-воркеров

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

	// для корректного завершения работы воркеров определяем контекст
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// организуем WaitGroup
	var wg sync.WaitGroup

	in := make(chan int, n)            // канал для передачи данных
	sigChan := make(chan os.Signal, 1) // sigChan канал для получения сигналов ОС, единичный буфер гарантирует вычитываение

	// регистрируем канал для получения сигналов прерывания программы пользователем (Ctrl+C)
	// или мягкого завершения окружением
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// запускаем заданное количество воркеров
	for i := 0; i < n; i++ {
		wg.Add(1) // увеличиваем счётчик WaitGroup
		go worker(ctx, in, &wg)
	}

	number := 0 // число для отправки в канал передачи данных

loop:
	for {
		select {
		case in <- number: // если можем, отправляем число в канал in
			number++
		case <-sigChan: // если же получаем сигнал прерывания, выходим из цикла
			break loop
		}
	}

	cancel()  // сигнал воркерам закругляться
	close(in) // закрываем in
	wg.Wait() // ждём остановки всех воркеров
}
