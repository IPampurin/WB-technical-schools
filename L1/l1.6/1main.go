/* ВАРИАНТ №1 - выход из горутины по условию */

package main

import (
	"fmt"
	"time"
)

const countForSearch = 5 // число, при нахождении которого завершаем горутину

// doSomething завершается по нахождению числа
func doSomething(in <-chan int, out chan<- string) {

	// слушаем канал in и печатаем числа из него
	for v := range in {
		time.Sleep(500 * time.Millisecond) // создаём вид бурной дейтельности
		fmt.Println(v)
		// если нашли искомое, завершаем работу
		if v == countForSearch {
			out <- fmt.Sprintf("Найдено искомое число %d, завершили горутину.", v)
			close(out)
			return
		}
	}
}

func main() {

	in := make(chan int)     // канал отправки данных в работу
	out := make(chan string) // канал приёма результатов

	// запускаем сравнение в отдельной горутине
	go doSomething(in, out)

	number := 0 // числа для отправки в работу

loop:
	for {
		select {
		case in <- number: // если можем, отправляем число в работу
			number++
		case v := <-out: // если в out появляется результат, выводим его и выходим из цикла
			fmt.Println(v)
			close(in)
			break loop
		}
	}

	fmt.Println("Завершили main().")
}
