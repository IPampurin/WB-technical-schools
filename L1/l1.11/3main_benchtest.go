/* Бенчмарк со сравнительным измерением времени для 1main.go и 2main.go */

package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"slices"
	"sync"
	"time"
)

// intersectionLine - линейная версия алгоритма
func intersectionLine(arrA, arrB []int) []int {

	numsMap := make(map[int]struct{})
	intersection := make([]int, 0)

	for _, a := range arrA {
		numsMap[a] = struct{}{}
	}

	for _, b := range arrB {
		if _, ok := numsMap[b]; ok {
			intersection = append(intersection, b)
			delete(numsMap, b)
		}
	}

	slices.Sort(intersection)
	return intersection
}

// intersectionPullWorkers - версия алгоритма с пулом воркеров
func intersectionPullWorkers(arrA, arrB []int) []int {

	numsMap := make(map[int]struct{})
	intersection := make([]int, 0)

	for _, a := range arrA {
		numsMap[a] = struct{}{}
	}

	numWorkers := runtime.NumCPU()
	bufferSize := numWorkers * 10

	var wg sync.WaitGroup
	var mu sync.Mutex

	job := make(chan int, bufferSize)
	result := make(chan int, bufferSize)
	done := make(chan struct{})

	go func() {
		for num := range result {
			intersection = append(intersection, num)
		}
		close(done)
	}()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for b := range job {
				mu.Lock()
				if _, ok := numsMap[b]; ok {
					delete(numsMap, b)
					result <- b
				}
				mu.Unlock()
			}
		}()
	}

	for _, b := range arrB {
		job <- b
	}
	close(job)

	wg.Wait()
	close(result)

	<-done

	slices.Sort(intersection)
	return intersection
}

// generateTestData - генератор тестовых массивов
func generateTestData(size int) ([]int, []int) {

	// организуем массивы для чисел
	// для упрощения примем размер массивов одинаковым
	arrA := make([]int, size)
	arrB := make([]int, size)

	// заполняем тестовые массивы числами
	for i := 0; i < size; i++ {
		arrA[i] = rand.Intn(size)
		arrB[i] = rand.Intn(size)
	}

	return arrA, arrB
}

func main() {

	rand.Seed(time.Now().UnixNano()) // определяем сид

	sizes := []int{1000, 10000, 50000, 100000, 500000, 1000000} // размеры тестовых массивов

	fmt.Println("\nСравнение времени выполнения:\n")
	fmt.Printf("%s    %s    %s\n", "Размер массивов", "Линейная версия", "Многопоточная версия")

	// производим замер для тестируемых размеров по очереди
	for _, size := range sizes {

		arrA, arrB := generateTestData(size) // генерируем массивы

		// замер времени линейной версии
		start := time.Now()                  // засекаем время
		res1 := intersectionLine(arrA, arrB) // запускаем функцию
		lineTime := time.Since(start)        // отмечаем сколько прошло времени

		// замер времени многопоточной версии
		start = time.Now()
		res2 := intersectionPullWorkers(arrA, arrB)
		pullTime := time.Since(start)

		// игнорируем результаты работы тестируемых функций
		_ = res1
		_ = res2

		fmt.Printf("%15d %15.4f ms %20.4f ms\n", size, lineTime.Seconds()*1000, pullTime.Seconds()*1000)
	}
}
