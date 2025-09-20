/* ВАРИАНТ №3 - решение с применением context.WithTimeout */

package main

import (
	"context"
	"fmt"
	"time"
)

const n = 3 // время работы программы в секундах

// consumer совершает работу с числами из канала
func consumer(ctx context.Context, ch <-chan int) {

	for {
		// постоянно слушаем два канала
		select {
		case v, ok := <-ch: // и если канал данных не закрыт, печатаем то, что пришло
			if !ok {
				return // если канал закрыт, завершаем работу
			}
			time.Sleep(500 * time.Millisecond) // создаём вид бурной деятельности
			fmt.Println(v)
		case <-ctx.Done():
			return // при получении сигнала по каналу отмены контекста завершаем работу
		}
	}
}

func main() {

	// создаём контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), n*time.Second)
	defer cancel()

	ch := make(chan int) // канал для отправки данных

	// запускаем чтение из канала
	go consumer(ctx, ch)

	number := 0 // число для отправки в канал передачи данных

loop:
	for {
		select {
		case ch <- number: // если можем, отправляем число в канал ch
			number++
		case <-ctx.Done(): // когда получаем сигнал
			close(ch)  // закрываем канал передачи данных
			break loop // и выходим из цикла
		}
	}

}
