/* ВАРИАНТ №3 - решение с sync.WaitGroup и хаотичным порядком вывода */

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
		go func(num int) {
			defer wg.Done()            // по закрытии горутины вычитаем счётчик WaitGroup
			fmt.Printf("%d ", num*num) // печатаем результат из горутины
		}(input[i])
	}
	// ждём окончания работы всех горутин. Вывод получаем в хаотичном порядке.
	wg.Wait()
}
