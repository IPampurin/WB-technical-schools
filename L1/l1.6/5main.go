/* ВАРИАНТ №5 - завершение работы с runtime.Goexit() */

package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

const countForSearch = 8 // число, при нахождении которого завершаем горутину

// doSomething удваивает числа
func doSomething(in <-chan int, out chan<- int, wg *sync.WaitGroup) {

	defer wg.Done()                          // вычитаем счетчик WaitGroup
	defer close(out)                         // гарантируем закрытие out
	defer fmt.Println("Завершили горутину.") // Goexit() дает выполниться всем defer-ам

	// слушаем канал
	for v := range in {
		time.Sleep(500 * time.Millisecond) // создаём вид бурной дейтельности
		out <- v * 2                       // отправляем удвоенное принятое число
		// проверяем не достигли ли искомого числа
		if v*2 == countForSearch {
			fmt.Println("Найдено искомое число, завершаем горутину через Goexit().")
			runtime.Goexit() // немедленное завершение горутины с учетом defer
		}
	}
}

func main() {

	in := make(chan int)  // канал отправки данных в работу
	out := make(chan int) // канал приёма результатов

	var wg sync.WaitGroup

	wg.Add(1)                    // добавляем счетчик WaitGroup
	go doSomething(in, out, &wg) // запускаем отдельную горутину

	number := 0 // числа для отправки в работу

loop:
	for {
		select {
		case in <- number: // если можем, отправляем число в работу
			number++
		case v, ok := <-out: // если в out появляется результат, выводим его и проверяем условие
			if !ok {
				break loop
			}
			fmt.Println(v)
			// если поступившее число равно искомому, закрываем канал и выходим из цикла
			if v == countForSearch {
				fmt.Println("Поступил искомый результат в main().")
				close(in)
				break loop
			}
		}
	}

	wg.Wait() // ждем завершения горутины

	fmt.Println("Завершили main().")
}
