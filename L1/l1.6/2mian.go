/* ВАРИАНТ №2 - выход из горутины по сигналу через канал уведомления */

package main

import (
	"fmt"
	"time"
)

const countForSearch = 8 // число, при нахождении которого завершаем горутину

// doSomething удваивает числа
func doSomething(in <-chan int, out chan<- int, done chan struct{}) {

	defer close(out) // гарантируем закрытие out

	for {
		select { // слушаем in
		case v, ok := <-in: // и если поступает число, удваиваем его
			if !ok {
				return
			}
			time.Sleep(500 * time.Millisecond) // создаём вид бурной дейтельности
			out <- v * 2                       // отправляем удвоенное принятое число
		case <-done: // при поступлении сигнала, закрываем out и завершаем горутину
			fmt.Println("Завершили горутину.")
			return
		}
	}
}

func main() {

	in := make(chan int)        // канал отправки данных в работу
	out := make(chan int)       // канал приёма результатов
	done := make(chan struct{}) // канал отмены

	// запускаем сравнение в отдельной горутине
	go doSomething(in, out, done)

	number := 0 // числа для отправки в работу

loop:
	for {
		select {
		case in <- number: // если можем, отправляем число в работу
			number++
		case v, ok := <-out: // если в out появляется результат, выводим его и проверяем условие
			if !ok {
				break loop
			}
			fmt.Println(v)
			// если поступившее число равно искомому, закрываем каналы и выходим из цикла
			if v == countForSearch {
				fmt.Println("Поступил искомый результат.")
				close(done)
				close(in)
				break loop
			}
		}
	}

	time.Sleep(1 * time.Millisecond) // возьмём паузу, чтобы горутина успела напечатать сообщение

	fmt.Println("Завершили main().")
}
