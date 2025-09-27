/* ВАРИАНТ №3 - группировка температур в мапу по ТЗ с конвейером и пулом воркеров */

// для столь скромной задачи асинхронная обработка избыточна
// и при таком объёме данных будет медленнее линейного варианта

package main

import (
	"fmt"
	"maps"
	"runtime"
	"slices"
	"sync"
)

type Element struct {
	key  int
	temp float32
}

// producer пишет входные данные из слайса в канал tempIn
func producer(tempIn chan<- float32, wg *sync.WaitGroup, temperatures []float32) {

	defer wg.Done()
	for _, v := range temperatures {
		tempIn <- v
	}
	close(tempIn)
}

// worker обрабатывает и структурирует данные для дальнейшей записи
func worker(tempIn <-chan float32, wg *sync.WaitGroup, elementOut chan<- Element) {

	defer wg.Done()
	for temp := range tempIn { // слушаем tempIn до его закрытия
		element := Element{
			key:  int(temp) / 10 * 10, // определяем десяток для группировки
			temp: temp,
		}
		elementOut <- element // отправляем структурированные данные
	}
}

func main() {

	// temperatures - входные данные
	temperatures := []float32{-25.4, -27.0, 13.0, 19.0, 15.5, 24.5, -21.0, 32.5}

	var wg sync.WaitGroup // WaitGroup тут не обязателен

	// количество воркеров ориентируем на количество ядер машины
	numWorkers := runtime.NumCPU()
	// размер буфера каналов в случае обработки больших объёмов данных
	bufferSize := numWorkers * 10

	tempIn := make(chan float32, bufferSize)     // канал для передачи данных на обработку
	elementOut := make(chan Element, bufferSize) // канал для передачи обработанных данных на запись

	// запускаем продюсер
	wg.Add(1)
	go producer(tempIn, &wg, temperatures)

	// создаём мапу (ключ - интервал, значение - слайс температур)
	groupsTemp := make(map[int][]float32, len(temperatures))

	// пул воркеров пишет в канал elementOut результат вычислений
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(tempIn, &wg, elementOut)
	}

	// ожидание горутин и закрытие канала реализуем в отдельной горутине
	go func() {
		wg.Wait()
		close(elementOut)
	}()

	for element := range elementOut { // слушаем elementOut до его закрытия
		groupsTemp[element.key] = append(groupsTemp[element.key], element.temp) // append работает со слайсом и без инициализации
	}

	// для наглядности выведем результат с сортировкой по ключам
	keys := slices.Sorted(maps.Keys(groupsTemp))
	for _, val := range keys {
		fmt.Printf("%3d : %v\n", val, groupsTemp[val])
	}
}
