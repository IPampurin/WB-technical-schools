/* ВАРИАНТ №2 - конкурентная запись в мапу с применением sync.Map */

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

	var uniq sync.Map // мапа, в которую будем писать данные

	var wg sync.WaitGroup

	// запускаем горутины - каждую букву входного слайса обрабатываем в отдельной горутине
	for _, let := range input {
		wg.Add(1)
		go func(letter string) {
			defer wg.Done()

			// методом LoadOrStore получаем значение count и параметр loaded; если значения не было, то loaded == false и
			// соответственно присваиваем count = 1, а если запись уже была в мапе, то loaded == true и методом Store
			// пишем в мапу новое значение - count++
			count, loaded := uniq.LoadOrStore(letter, 1)
			if loaded {
				// в sync.Map в качестве ключа, значения используется any, поэтому при добавлении единицы привдим тип
				uniq.Store(letter, count.(int)+1)
			}

		}(let)
	}

	wg.Wait() // ждем пока все горутины закончат запись в мапу

	// собираем результаты для сортировки и красивого вывода в отдельную мапу
	justUniq := make(map[string]int)

	// метод Range перебирает все элементы sync.Map и для каждого вызывает
	// функцию f (если нужно прекратить перебор, то функция должна возвратить false)
	uniq.Range(func(key, value interface{}) bool {
		justUniq[key.(string)] = value.(int)
		return true
	})

	keys := slices.Sorted(maps.Keys(justUniq)) // получаем отсортированный слайс ключей мапы

	// форматировано выводим содержимое мапы
	for _, v := range keys {
		fmt.Printf("key = %s, val = %d\n", v, justUniq[v])
	}
}
