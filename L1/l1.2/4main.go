/* ВАРИАНТ №4 - решение с sync.WaitGroup и корректным порядком вывода */

package main

import (
	"fmt"
	"sync"
)

func main() {

	// входные данные
	input := []int{2, 4, 6, 8, 10}

	// задаём WaitGroup
	var wg sync.WaitGroup

	// запускаем len(input) горутин
	for i := 0; i < len(input); i++ {
		wg.Add(1) // прибавляем счётчик WaitGroup
		go func(idx int) {
			defer wg.Done()                      // по закрытии горутины вычитаем счётчик WaitGroup
			input[idx] = input[idx] * input[idx] // меняем значения в исходном массиве
		}(i)
	}
	// ждём окончания работы всех горутин
	wg.Wait()

	// печатаем элементы изменённого слайса
	for _, v := range input {
		fmt.Printf("%d ", v)
	}
	// стоит отметить, что race condition в данном случае избегаем за счёт передачи
	// в работу горутинам конкретных элементов слайса (через индекс элемента)
}
