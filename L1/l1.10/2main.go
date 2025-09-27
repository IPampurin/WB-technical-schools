/* ВАРИАНТ №2 - асинхронная группировка температур в мапу по ТЗ */

// для столь скромной задачи асинхронная обработка избыточна
// и при таком объёме данных будет медленнее линейного варианта

package main

import (
	"fmt"
	"maps"
	"slices"
	"sync"
)

func main() {

	// temperatures - входные данные
	temperatures := []float32{-25.4, -27.0, 13.0, 19.0, 15.5, 24.5, -21.0, 32.5}

	// создаём мапу (ключ - интервал, значение - слайс температур)
	groupsTemp := make(map[int][]float32, len(temperatures))

	var wg sync.WaitGroup
	var mu sync.Mutex

	// запускаем len(temperatures) горутин, чтобы каждое значение из входных данных
	// обрабатывалось отдельно, и добавляем температуры в значения соответствующего интервала
	for i := 0; i < len(temperatures); i++ {
		wg.Add(1)
		go func(i int) { // передавать wg и mu нет необходимости, так как тут горутины - замыкания
			defer wg.Done()
			key := int(temperatures[i]) / 10 * 10 // определяем десяток для группировки
			mu.Lock()
			groupsTemp[key] = append(groupsTemp[key], temperatures[i]) // append работает со слайсом и без инициализации
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// для наглядности выведем результат с сортировкой по ключам
	keys := slices.Sorted(maps.Keys(groupsTemp))
	for _, val := range keys {
		fmt.Printf("%3d : %v\n", val, groupsTemp[val])
	}
}
