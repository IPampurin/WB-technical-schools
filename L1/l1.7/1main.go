/* ВАРИАНТ №1 - конкурентная запись в мапу с применением sync.Mutex */

package main

import (
	"fmt"
	"maps"
	"math/rand"
	"slices"
	"sync"
)

// letterGenerator выдаёт случайную букву из набора
func letterGenerator() string {

	letters := []rune("abcdefghijklmnopqrstuvwxyz") // возможный набор символов
	idx := rand.Intn(len(letters))                  // определяем индекс руны
	return string(letters[idx])
}

// genInput служит для получения слайса букв как входных данных
func genInput(n int) []string {

	input := make([]string, n, n) // слайс букв для записи в мапу и подсчёта

	var wg sync.WaitGroup

	// для заполнения слайса запускаем n горутин
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			input[i] = letterGenerator()
		}(i)
	}
	wg.Wait() // дожидаемся заполнения

	return input
}

func main() {

	n := 240 // количество букв, которые будем записывать в мапу и считать их количество (произвольное значение)

	input := genInput(n)

	uniq := make(map[string]int, len(input)) // мапа, в которую будем писать данные

	var wg sync.WaitGroup
	var mu sync.Mutex

	// запускаем горутины - каждую букву входного слайса обрабатываем в отдельной горутине
	for i := 0; i < len(input); i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			mu.Lock() // блокируем операции с мапой всеми другими горутинами
			if _, ok := uniq[input[i]]; !ok {
				uniq[input[i]] = 1
			} else {
				uniq[input[i]]++
			}
			mu.Unlock() // открываем мапу для других горутин
		}(i)
	}

	wg.Wait() // ждем пока все горутины закончат запись в мапу

	keys := slices.Sorted(maps.Keys(uniq)) // получаем отсортированный слайс ключей мапы

	// форматировано выводим содержимое мапы
	for _, v := range keys {
		fmt.Printf("key = %s, val = %d\n", v, uniq[v])
	}
}
