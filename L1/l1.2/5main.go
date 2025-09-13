/* ВАРИАНТ №5 - решение с использованием буферизированного канала и хаотичным порядком вывода */

package main

import (
	"fmt"
)

func main() {

	// входные данные
	input := []int{2, 4, 6, 8, 10}

	ch := make(chan int, len(input))

	// задаём WaitGroup
	//var wg sync.WaitGroup

	// запускаем len(input) горутин
	for i := 0; i < len(input); i++ {
		//	wg.Add(1)
		go func(idx int) {
			//		defer wg.Done()
			ch <- input[idx] // отправляем элемент в канал
		}(i)
	}
	//wg.Wait()

	close(ch) // закрываем канал

	// печатаем квадрат числа, вычитанного из канала
	for v := range ch {
		fmt.Printf("%d ", v*v)
	}
}
