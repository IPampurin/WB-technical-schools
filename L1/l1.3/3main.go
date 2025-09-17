/* ВАРИАНТ №2 - решение с применением паттерна на основе канала done */

package main

import (
	"fmt"
	"sync"
	"time"
)

const (
	n         = 5 // необходимое количество воркеров
	limitTime = 3 // время выдачи данных в секундах
)

// worker печатает в консоль значение, полученное из канала
func worker(ch chan any, done chan struct{}, wg sync.WaitGroup) {

	defer wg.Done()

	for {
		// постоянно слушаем два канала
		select {
		case v, ok := <-ch: // и если основной канал не закрыт, печатаем то, что пришло
			if !ok {
				return // если канал закрыт, завершаем работу
			}
			fmt.Println(v)
		case <-done:
			return // при получении сигнала по каналу отмены завершаем worker
		}
	}
}

func main() {

	var wg sync.WaitGroup

	done := make(chan struct{})
	in := make(chan any, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go worker(in, done, wg)
	}

	number := 0

	for {
		select {
		case in <- number:
			number++
		case <-time.After(limitTime * time.Second):
			close(done)
			close(in)
			break
		}
	}

}
