/* ВАРИАНТ №7 - решение с каналом отмены и с корректным порядком вывода */

package main

import "fmt"

func main() {

	// входные данные
	input := []int{2, 4, 6, 8, 10}

	// создаём канал, в который будем слать сигнал по завершении горутины
	done := make(chan struct{})

	// создаём слайс для квадратов чисел
	results := make([]int, len(input))

	// запускаем len(input) горутин
	for i, num := range input {
		go func(idx int, n int) {
			results[idx] = n * n // пишем в слайс квадрат числа
			done <- struct{}{}   // отправляем сигнал
		}(i, num)
	}

	// ожидаем завершения всех горутин
	for i := 0; i < len(input); i++ {
		<-done
	}

	// выводим результат
	for _, v := range results {
		fmt.Printf("%d ", v)
	}
}
