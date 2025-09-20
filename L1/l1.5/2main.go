/* ВАРИАНТ №2 - решение с применением time.Timer */

package main

import (
	"fmt"
	"time"
)

const n = 3 // время работы программы в секундах

// consumer совершает работу с числами из канала
func consumer(ch <-chan int, done chan struct{}) {

	for {
		// постоянно слушаем два канала
		select {
		case v, ok := <-ch: // и если канал данных не закрыт, печатаем то, что пришло
			if !ok {
				return // если канал закрыт, завершаем работу
			}
			time.Sleep(500 * time.Millisecond) // создаём вид бурной деятельности
			fmt.Println(v)
		case <-done:
			return // при получении сигнала по каналу отмены завершаем работу
		}
	}
}

func main() {

	// засекаем время.
	// инициализация timer вынесена за пределы цикла с select,
	// чтобы при каждой итерации не получать новый таймер
	timer := time.NewTimer(n * time.Second)

	ch := make(chan int)        // канал для отправки данных
	done := make(chan struct{}) // канал отмены

	// запускаем чтение из канала
	go consumer(ch, done)

	number := 0 // число для отправки в канал передачи данных

loop:
	for {
		select {
		case ch <- number: // если можем, отправляем число в канал ch
			number++
		case <-timer.C: // когда приходит время - выходим из цикла
			close(done) // сначала закрываем done, чтобы его можно было вычитать в consumer()
			close(ch)   // потом закрываем канал передачи данных
			break loop
		}
	}

}
