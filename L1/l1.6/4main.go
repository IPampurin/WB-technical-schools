/* ВАРИАНТ №4 - выход из горутинв через контекст с таймаутом */

package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const timeLimit = 3 // время работы программы в секундах

// doSomething удваивает числа
func doSomething(ctx context.Context, in <-chan int, out chan<- int, wg *sync.WaitGroup) {

	defer wg.Done()  // вычитаем счетчик WaitGroup
	defer close(out) // гарантируем закрытие out

	for {
		select { // слушаем in
		case v, ok := <-in: // и если поступает число, удваиваем его
			if !ok {
				return
			}
			time.Sleep(500 * time.Millisecond) // создаём вид бурной дейтельности
			// но из-за этой паузы возникает ситуация, когда main() уже не слушает out и можем
			// получить dedlock при отправке результата. Поэтому перед отправкой выполяем проверку
			select {
			case out <- v * 2: // если это возможно, отправляем результат
			case <-ctx.Done(): // а если время вышло, то закрываем канал и выходим
				fmt.Println("Завершили горутину.")
				return
			}
		case <-ctx.Done(): // при поступлении сигнала завершаем горутину
			fmt.Println("Завершили горутину.")
			return
		}
	}
}

func main() {

	// определяем контекст
	ctx, cancel := context.WithTimeout(context.Background(), timeLimit*time.Second)
	defer cancel() // гарантируем отмену контекста

	var wg sync.WaitGroup

	in := make(chan int)  // канал отправки данных в работу
	out := make(chan int) // канал приёма результатов

	wg.Add(1)                         // добавляем счётчик WaitGroup
	go doSomething(ctx, in, out, &wg) // запускаем отдельную горутину

	number := 0 // числа для отправки в работу

loop:
	for {
		select {
		case in <- number: // если можем, отправляем число в работу
			number++
		case v, ok := <-out: // если в out появляется результат, выводим его и проверяем условие
			if !ok {
				break loop // если out закрыт, выходим из цикла
			}
			fmt.Println(v)
		case <-ctx.Done(): // при отмене контекста
			close(in)  // закрываем in
			break loop // выходим из цикла
		}
	}

	wg.Wait() // ждем завершения горутины

	fmt.Println("Завершили main().")
}
