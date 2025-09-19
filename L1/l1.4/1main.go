/* ВАРИАНТ №3 - решение с применением паттерна на основе канала done и остановкой записи по времени */

package main

import (
	"fmt"
	"sync"
	"time"
)

const (
	n         = 5 // необходимое количество воркеров
	limitTime = 5 // время выдачи данных в секундах
)

// worker печатает в консоль значение, полученное из канала
func worker(ch chan int, done chan struct{}, wg *sync.WaitGroup) {

	defer wg.Done() // уменьшаем счётчик WaitGroup
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

	// организуем WaitGroup
	var wg sync.WaitGroup

	done := make(chan struct{}) // канал отмены
	in := make(chan int, n)     // канал для передачи данных

	// запускаем n воркеров
	for i := 0; i < n; i++ {
		wg.Add(1) // увеличиваем счётчик WaitGroup
		go worker(in, done, &wg)
	}

	// для корректной остановки горутин через limitTime секунд закроем канал отмены
	time.AfterFunc(limitTime*time.Second, func() {
		close(done)
	})

	number := 0 // число для отправки в канал передачи данных

loop:
	for {
		select {
		// если можем, отправляем число в канал in
		case in <- number:
			number++
			// если же done закрыт и можем его вычитать, закрываем in и выходим из цикла
		case <-done:
			close(in)
			break loop
		}
	}

	// ждём остановки всех воркеров
	wg.Wait()
}
