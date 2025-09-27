/* ВАРИАНТ №1 - линейная группировка температур в мапу по ТЗ */

package main

import (
	"fmt"
	"maps"
	"slices"
)

func main() {

	// temperatures - входные данные
	temperatures := []float32{-25.4, -27.0, 13.0, 19.0, 15.5, 24.5, -21.0, 32.5}

	// создаём мапу (ключ - интервал, значение - слайс температур)
	groupsTemp := make(map[int][]float32, len(temperatures))

	// проходим по input и добавляем температуры в значения соответствующего интервала
	for _, temp := range temperatures {
		key := int(temp) / 10 * 10                      // определяем десяток для группировки
		groupsTemp[key] = append(groupsTemp[key], temp) // append работает со слайсом и без инициализации
	}

	// для наглядности выведем результат с сортировкой по ключам
	keys := slices.Sorted(maps.Keys(groupsTemp))
	for _, val := range keys {
		fmt.Printf("%3d : %v\n", val, groupsTemp[val])
	}
}
