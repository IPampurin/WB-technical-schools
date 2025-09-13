/* ВАРИАНТ №6 - решение без sync.WaitGroup и с хаотичным порядком вывода */

package main

import "fmt"

func main() {

	// входные данные
	input := []int{2, 4, 6, 8, 10}

	// создаём канал, куда будем писать элементы из input
	jobChan := make(chan int, len(input))

	// создаём канал, из которого будем забирать квадраты чисел
	resChan := make(chan int, len(input))

	// запускаем len(input) горутин (пул воркеров)
	for i := 0; i < len(input); i++ {
		go func(jobs <-chan int, res chan<- int) {
			for v := range jobs {
				res <- v * v
			}
		}(jobChan, resChan)
	}

	// отправляем числа из массива в канал jobChan
	for i := 0; i < len(input); i++ {
		jobChan <- input[i]
	}
	// закрываем канал jobChan
	close(jobChan)

	// получаем len(input) чисел из канала с результатами и печатаем их
	for i := 0; i < len(input); i++ {
		fmt.Printf("%d ", <-resChan)
	}
	// для порядка закрываем канал resChan
	close(resChan)
}
