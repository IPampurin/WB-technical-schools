/* ВАРИАНТ №2 - нахождение пересечения двух массивов пулом воркеров */

// Примечание: версия для больших объёмов данных (>= 10_000)
// с малым количеством данных линейная версия должна быть быстрее

package main

import (
	"fmt"
	"runtime"
	"slices"
	"sync"
)

func main() {

	arrA := []int{1, 2, 3, 1, 2, 3, 6}          // первый входной массив
	arrB := []int{6, 6, 6, 2, 3, 4, 15, 15, 25} // второй входной массив

	numsMap := make(map[int]struct{}) // мапа для хранения уникальных значений первого массива
	intersection := make([]int, 0)    // массив для отражения пересекающихся данных

	// итерируемся по первому массиву и вносим значения из него в мапу как ключи
	for _, a := range arrA {
		numsMap[a] = struct{}{}
	}

	// количество воркеров ориентируем на количество ядер машины
	numWorkers := runtime.NumCPU()
	// назначим размер буфера каналов в случае обработки больших объёмов данных
	bufferSize := numWorkers * 10

	var wg sync.WaitGroup
	var mu sync.Mutex

	job := make(chan int, bufferSize)    // канал для передачи воркерам значений из второго массива
	result := make(chan int, bufferSize) // канал для возврата числа, входящего в пересечение

	// запускаем numWorkers воркеров
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for b := range job { // слушаем канал job
				mu.Lock()
				if _, ok := numsMap[b]; ok {
					delete(numsMap, b) // если значение в мапе есть, удаляем его, чтобы не дублировать в данных о пересечении
					result <- b        // отправляем число в канал для последующей записи в массив с пересечением
				}
				mu.Unlock()
			}
		}()
	}

	// отправляем числа из второго массива воркерам в работу
	for _, b := range arrB {
		job <- b
	}
	close(job) // закрываем job, чтобы воркеры закруглялись

	// ждём завершения воркеров и закрываем канал с результатами
	go func() {
		wg.Wait()
		close(result)
	}()

	// вычитываем result пока в нём что-то будет и пишем числа в массив с пересекающимися данными
	for num := range result {
		intersection = append(intersection, num)
	}

	slices.Sort(intersection) // отсортируем для красоты (не обязательно)

	fmt.Println("intersection =", intersection) // выводим результат
}
