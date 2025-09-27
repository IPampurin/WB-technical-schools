/* ВАРИАНТ №2 - пример конвейера чисел с разделением на отдельные функции */

package main

import (
	"fmt"
	"sync"
)

// producer пишет по порядку числа из input в канал in
func producer(in chan<- int, wg *sync.WaitGroup, nums []int) {

	defer wg.Done()
	for _, v := range nums {
		in <- v
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
		fmt.Printf("%d ", v)
	}
}

func main() {

	// input - входные данные
	input := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	in := make(chan int)  // канал, в который пишем числа из массива
	out := make(chan int) // канал, из которого забираем удвоенные числа

	var wg sync.WaitGroup

	wg.Add(1)
	go producer(in, &wg, input) // отправляем числа в работу

	wg.Add(1)
	go transformer(in, out, &wg) // удваиваем числа и отправляем на вывод

	wg.Add(1)
	go consumer(out, &wg) // выводим результат

	wg.Wait() // блокируем main до завершения горутин
}
