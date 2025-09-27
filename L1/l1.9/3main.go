/* ВАРИАНТ №3 - пример конвейера чисел с буферизированными каналами */

// Примечание:
// - теоретический прирост производительности при больших объёмах данных на 10-30%
// - буферизация каналов логична, если скорость обработки данных в горутинах разная,
//	 также размер буфера логично подбирать под конкретные задачи программы

package main

import (
	"fmt"
	"sync"
)

const (
	n     = 1000000 // количество чисел на отправку в работу
	filtr = 100000  // чтобы не захламлять вывод, будем печатать первое и каждое filtr-е число
)

// producer пишет по порядку числа в канал in
func producer(in chan<- int, wg *sync.WaitGroup) {

	defer wg.Done()
	for i := 1; i <= n; i++ {
		in <- i
	}
	close(in) // завершаем запись закрытием канала
}

// transformer слушает канал in и пишет результат в канал out
func transformer(in <-chan int, out chan<- int, wg *sync.WaitGroup) {

	defer wg.Done()
	for v := range in {
		out <- v * 2
	}
	close(out) // завершаем запись закрытием канала
}

// consumer слушает out пока в нём есть числа и печатает результат
func consumer(out <-chan int, wg *sync.WaitGroup) {

	defer wg.Done()
	for v := range out {
		if v == 2 || v%filtr == 0 { // для наглядности выводим числа с интервалом
			fmt.Printf("%d ", v)
		}
	}
}

func main() {

	in := make(chan int, 1000)  // канал, в который пишем числа
	out := make(chan int, 1000) // канал, из которого забираем удвоенные числа

	var wg sync.WaitGroup

	wg.Add(1)
	go producer(in, &wg) // отправляем числа в работу

	wg.Add(1)
	go transformer(in, out, &wg) // удваиваем числа и отправляем на вывод

	wg.Add(1)
	go consumer(out, &wg) // выводим результат

	wg.Wait() // блокируем main до завершения горутин
}
