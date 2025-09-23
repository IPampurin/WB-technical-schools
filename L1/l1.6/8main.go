/* ВАРИАНТ №8 - остановка горутины через флаг sync/atomic (так себе вариант) */

package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const countForSearch = 8 // число, при нахождении которого завершаем горутину

// doSomething удваивает числа
func doSomething(in <-chan int, out chan<- int, stopFlag *int32, wg *sync.WaitGroup) {

	defer wg.Done()
	defer close(out) // гарантируем закрытие out

	for {
		// проверяем флаг остановки
		if atomic.LoadInt32(stopFlag) == 1 {
			fmt.Println("Завершили горутину по флагу перед работой.")
			return
		}

		select {
		case v, ok := <-in:
			if !ok {
				// канал закрыт, но мы игнорируем это и ждём флаг
				continue
			}

			time.Sleep(500 * time.Millisecond) // создаём вид бурной дейтельности

			// проверяем флаг перед отправкой
			if atomic.LoadInt32(stopFlag) == 1 {
				fmt.Println("Завершили горутину по флагу перед отправкой.")
				return
			}
			out <- v * 2 // отправляем удвоенное принятое число
		}
	}
}

func main() {

	in := make(chan int, 1)  // канал отправки данных в работу
	out := make(chan int, 1) // канал приёма результатов
	var stopFlag int32       // атомарный флаг (0 - работает, 1 - остановка)

	var wg sync.WaitGroup // используем WaitGroup чтобы дождаться горутину
	wg.Add(1)
	// запускаем отдельную горутину
	go doSomething(in, out, &stopFlag, &wg)

	number := 0 // числа для отправки в работу

loop:
	for {

		// проверяем флаг остановки
		if atomic.LoadInt32(&stopFlag) == 1 {
			break loop
		}

		select {
		case in <- number: // если можем, отправляем число в работу
			number++
		case v, ok := <-out: // если в out появляется результат, выводим его и проверяем условие
			if !ok {
				break loop
			}
			fmt.Println(v)
			// если поступившее число равно искомому, устанавливаем флаг и выходим из цикла
			if v == countForSearch {
				fmt.Println("Поступил искомый результат.")
				atomic.StoreInt32(&stopFlag, 1)
				close(in)
				break loop
			}
		}
	}

	wg.Wait() // ждём горутину

	fmt.Println("Завершили main().")
}
