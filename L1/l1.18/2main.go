/* ВАРИАНТ №2 - решение с применением sync.Mutex */

package main

import (
	"fmt"
	"sync"
	"time"
)

// worker имитирует работу и увеличивает счётчик
func worker(id int, chIn <-chan int, wg *sync.WaitGroup, mu *sync.Mutex, num *int) {

	defer wg.Done()
	// слушаем chIn до его закрытия
	for v := range chIn {
		time.Sleep(50 * time.Millisecond) // создаём вид бурной деятельности
		mu.Lock()                         // блокируем инкрементацию и вывод, чтобы снять конкуренцию в stdout
		*num++                            // увеличиваем счётчик на единицу и печатаем результат
		fmt.Printf("воркер %2.d обработал значение %2.d (результат: %3.d) и инкрементировал счётчик (значение: %2.d)\n", id, v, v*v, *num)
		mu.Unlock() // снимаем блокировку
	}
}

func main() {

	// input слайс чисел, операции над которыми будем подсчитывать в конкурентной среде
	input := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	ch := make(chan int) // канал раздачи данных воркерам
	var counter int      // счётчик операций с числами в конкурентной среде

	var wg sync.WaitGroup
	var mu sync.Mutex

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
		go worker(i, ch, &wg, &mu, &counter)
	}

	wg.Wait() // ждём окончания работы воркеров

	// выводим результат
	fmt.Printf("\nОбработано чисел: %d\n", counter)
}
