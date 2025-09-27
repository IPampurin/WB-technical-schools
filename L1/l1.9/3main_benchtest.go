/* Бенчмарк с измерением времени и памяти для 3main.go (ВАРИАНТ №3 - пример конвейера чисел с буферизированными каналами) */

package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

const (
	n     = 1000000 // количество чисел на отправку в работу
	filtr = 100000  // чтобы не захламлять вывод, будем печатать первое и каждое filtr-е число
)

// producer пишет по порядку числа в канал in
func producer(in chan<- int, wg *sync.WaitGroup) {

	defer wg.Done()
	for i := 1; i <= n; i++ {
		in <- i
	}
	close(in) // завершаем запись закрытием канала
}

// transformer слушает канал in и пишет результат в канал out
func transformer(in <-chan int, out chan<- int, wg *sync.WaitGroup) {

	defer wg.Done()
	for v := range in {
		out <- v * 2
	}
	close(out) // завершаем запись закрытием канала
}

// consumer слушает out пока в нём есть числа и печатает результат
func consumer(out <-chan int, wg *sync.WaitGroup) {

	defer wg.Done()
	for v := range out {
		if v == 2 || v%filtr == 0 { // для наглядности выводим числа с интервалом
			_ = v // игнорируем вывод
		}
	}
}

// runPipelineWithMem реализует логику нашего конвейера и измеряет время и выделение памяти
func runPipelineWithMem(bufferSize int) (time.Duration, uint64) {

	var m1, m2 runtime.MemStats // создаём две переменные для статистики памяти
	runtime.GC()                // принудительно запускаем сборщик мусора
	runtime.ReadMemStats(&m1)   // считываем начальную статистику памяти

	start := time.Now() // засекаем время

	in := make(chan int, bufferSize)  // канал, в который пишем числа
	out := make(chan int, bufferSize) // канал, из которого забираем удвоенные числа

	var wg sync.WaitGroup

	wg.Add(1)
	go producer(in, &wg) // отправляем числа в работу

	wg.Add(1)
	go transformer(in, out, &wg) // удваиваем числа и отправляем на вывод

	wg.Add(1)
	go consumer(out, &wg) // выводим результат

	wg.Wait() // блокируем main до завершения горутин

	duration := time.Since(start) // смотрим сколько времени прошло со start

	runtime.ReadMemStats(&m2)                   // считываем текущую статистику памяти
	memoryUsed := m2.TotalAlloc - m1.TotalAlloc // вычисляем сколько памяти прошло через наш код (по куче, стек не измеряем)

	return duration, memoryUsed
}

func main() {

	bufferSizes := []int{0, 1, 10, 100, 1000, 10000, 100000} // размеры буферов для теста

	fmt.Printf("\nТестирование производительности с %d элементами:\n", n)
	fmt.Println()
	fmt.Printf("%8s    %-10s    %8s\n", "Буфер", "Время", "Память")
	fmt.Println()

	// перебираем размеры буферов
	for _, size := range bufferSizes {

		iterations := 3                 // количество измерений для одного буфера, для усреднения результатов
		var totalDuration time.Duration // суммарное время на все итерации
		var totalMemory uint64          // суммарная память на все итерации

		for i := 0; i < iterations; i++ {
			duration, memory := runPipelineWithMem(size)
			totalDuration += duration // суммируем время очередного прогона
			totalMemory += memory     // суммируем память очередного прогона
		}

		// вычисляем средние время и память по всем прогонам для одного буфера
		avgDuration := totalDuration / time.Duration(iterations) // среднее время
		avgMemory := totalMemory / uint64(iterations)            // среднее выделение памяти

		msec := avgDuration.Seconds() * 1000 // время выводить будем в миллисекундах

		fmt.Printf("%8d    %8.4f ms   %8d байт\n",
			size, msec, avgMemory)
	}
}
