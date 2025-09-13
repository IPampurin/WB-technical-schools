/* ВАРИАНТ №5 - решение с использованием буферизированного канала и хаотичным порядком вывода */

package main

import (
	"fmt"
	"sync"
)

func main() {

	// входные данные
	input := []int{2, 4, 6, 8, 10}

	// создаём канал с буфером по длине входных данных
	ch := make(chan int, len(input))

	// задаём WaitGroup
	var wg sync.WaitGroup

	// запускаем len(input) горутин
	for i := 0; i < len(input); i++ {
		wg.Add(1) // прибавляем счётчик WaitGroup
		go func() {
			defer wg.Done() // по закрытии горутины вычитаем счётчик WaitGroup
			// смотрим, что есть в канале и печатаем квадрат этого числа
			for v := range ch {
				fmt.Printf("%d ", v*v)
			}
		}()
	}

	// отправляем числа из массива в канал
	for i := 0; i < len(input); i++ {
		ch <- input[i]
	}

	// закрываем канал
	close(ch)
	// ждём окончания работы всех горутин
	wg.Wait()
}
