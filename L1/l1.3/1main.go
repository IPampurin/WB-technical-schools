/* ВАРИАНТ №1 - некорректное решение (числа выводятся, но нет контроля остановки горутин) */

package main

import (
	"fmt"
)

const n = 5 // необходимое количество воркеров

// worker слушает канал и печатает то, что приходит
func worker(ch chan any) {

	for v := range ch {
		fmt.Println(v)
	}
}

func main() {

	in := make(chan any) // создаём небуферизированный канал

	for i := 0; i < n; i++ {
		go worker(in)
	}

	for i := 0; i < 100000; i++ {
		in <- i
	}
}
