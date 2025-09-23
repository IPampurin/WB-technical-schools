/* ВАРИАНТ №9 - остановка горутины через panic (так себе вариант) */

package main

import (
	"fmt"
	"sync"
	"time"
)

const countForSearch = 7 // число, при нахождении которого завершаем горутину

// doSomething удваивает поступающие числа
func doSomething(in <-chan int, out chan<- int, wg *sync.WaitGroup) {

	defer wg.Done()  // вычитаем счетчик WaitGroup
	defer close(out) // закрываем out в любом случае
	// в случае паники перехватываем её и возвращаем значение ошибки, переданное при вызове panic
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Горутина восстановлена после паники: %v\n", r)
		}
	}()

	// слушаем канал in до его закрытия
	for v := range in {
		// если пришедшее число больше искомого, имитируем панику
		if v > countForSearch/2 {
			panic("изображаем панику.")
		}
		time.Sleep(500 * time.Millisecond) // создаём вид бурной дейтельности
		out <- v * 2                       // отправляем в out результат
	}

	fmt.Println("Горутина завершила работу.")
}

func main() {

	in := make(chan int)  // канал отправки данных в работу
	out := make(chan int) // канал приёма результатов

	var wg sync.WaitGroup

	wg.Add(1)                    // добавляем счетчик WaitGroup
	go doSomething(in, out, &wg) // запускаем сравнение в отдельной горутине

	number := 0 // числа для отправки в работу

loop:
	for {
		select {
		case in <- number: // если можем, отправляем число в работу
			number++
		case v, ok := <-out: // если в out появляется результат, выводим его и проверяем условие
			if !ok {
				break loop // если out закрыт, выходим из цикла
			}
			fmt.Println(v)
			// если встретили искомое число закрываем in и выходим из цикла
			if v == countForSearch {
				close(in)
				fmt.Println("Поступил искомый результат. Закрываем канал in.")
				break loop
			}
		}
	}

	wg.Wait()

	fmt.Println("Завершили main().")
}
