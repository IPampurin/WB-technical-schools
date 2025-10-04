/* ВАРИАНТ №1 - решение с применением sync/atomic */

package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// worker имитирует работу и увеличивает счётчик
func worker(id int, chIn <-chan int, wg *sync.WaitGroup, num *atomic.Int64) {

	defer wg.Done()
	// слушаем chIn до его закрытия
	for v := range chIn {
		time.Sleep(50 * time.Millisecond) // создаём вид бурной деятельности
		num.Add(1)                        // увеличиваем счётчик на единицу
		fmt.Printf("воркер %2.d обработал значение %2.d (результат: %3.d) и инкрементировал счётчик (значение: %2.d)\n", id, v, v*v, num.Load())
	}
}

func main() {

	// input слайс чисел, операции над которыми будем подсчитывать в конкурентной среде
	input := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	ch := make(chan int)     // канал раздачи данных воркерам
	var counter atomic.Int64 // атомарный счётчик

	var wg sync.WaitGroup

	// отправляем числа из слайса в канал
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, v := range input {
			ch <- v
		}
		close(ch) // закрываем канал, когда данные заканчиваются
	}()

	// запускаем четыре воркера
	for i := 1; i <= 4; i++ {
		wg.Add(1)
		go worker(i, ch, &wg, &counter)
	}

	wg.Wait() // ждём окончания работы воркером

	// выводим результат
	fmt.Printf("\nОбработано чисел: %d\n", counter.Load())
}
