/* ВАРИАНТ №4 - решение с применением контекста и остановкой записи по времени */

package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const (
	n         = 5 // необходимое количество воркеров
	limitTime = 3 // время выдачи данных в секундах
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

	// для корректного завершения работы через limitTime секунд закроем канал отмены контекста
	ctx, cancel := context.WithTimeout(context.Background(), limitTime*time.Second)
	defer cancel()

	// организуем WaitGroup
	var wg sync.WaitGroup

	in := make(chan int, n) // канал для передачи данных

	// запускаем n воркеров
	for i := 0; i < n; i++ {
		wg.Add(1) // увеличиваем счётчик WaitGroup
		go worker(ctx, in, &wg)
	}

	number := 0 // число для отправки в канал передачи данных

loop:
	for {
		select {
		// если можем, отправляем число в канал in
		case in <- number:
			number++
			// если же done закрыт и можем его вычитать, закрываем in и выходим из цикла
		case <-ctx.Done():
			close(in)
			break loop
		}
	}

	// ждём остановки всех воркеров
	wg.Wait()
}
