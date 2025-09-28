/* ВАРИАНТ №1 - линейное нахождение пересечения двух массивов */

package main

import (
	"fmt"
	"slices"
)

func main() {

	arrA := []int{1, 2, 3, 1, 2, 3, 6}          // первый входной массив
	arrB := []int{6, 6, 6, 2, 3, 4, 15, 15, 25} // второй входной массив

	numsMap := make(map[int]struct{}) // мапа для хранения уникальных значений
	intersection := make([]int, 0)    // массив для отражения пересекающихся данных

	// итерируемся по первому массиву и вносим значения из него в мапу как ключи
	for _, a := range arrA {
		numsMap[a] = struct{}{}
	}

	// итерируемся по второму массиву и проверяем наличие значений в мапе
	// если такое значение в мапе фигурирует, вносим его в пересечение и
	// удаляем его из мапы, чтобы не дублировать значения
	for _, b := range arrB {
		if _, ok := numsMap[b]; ok {
			intersection = append(intersection, b)
			delete(numsMap, b)
		}
	}

	slices.Sort(intersection) // отсортируем для красоты (не обязательно)

	fmt.Println("intersection =", intersection) // выводим результат
}
