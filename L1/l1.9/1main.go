/* ВАРИАНТ №1 - пример конвейера чисел (паттерн producer-transformer-consumer) */

package main

import "fmt"

func main() {

	// input - входные данные
	input := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	in := make(chan int)  // канал, в который пишем числа из массива
	out := make(chan int) // канал, из которого забираем удвоенные числа

	// в первой горутине пишем по порядку числа из input в канал in
	go func() {
		for _, v := range input {
			in <- v
		}
		close(in) // завершаем запись закрытием канала
	}()

	// во второй горутине слушаем канал in и пишем результат в канал out
	go func() {
		for v := range in {
			out <- v * 2
		}
		close(out) // завершаем запись закрытием канала
	}()

	//	в главной горутине слушаем out пока в нём есть числа и печатаем результат
	for v := range out {
		fmt.Printf("%d ", v)
	}
}
