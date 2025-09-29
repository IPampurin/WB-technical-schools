/* ВАРИАНТ №1 - нахождение уникальных слов с помощью мапы */

package main

import (
	"fmt"
	"maps"
	"slices"
)

func main() {

	input := []string{"cat", "cat", "dog", "cat", "tree"} // входные строки

	// мапа для уникальных строк как ключей
	// структура-значение чтобы не занимать память
	uniq := make(map[string]struct{}, len(input))

	// читаем input и пишем в мапу - останутся только уникальные ключи
	for _, v := range input {
		uniq[v] = struct{}{}
	}

	// получаем сортированный слайс ключей мапы
	keys := slices.Sorted(maps.Keys(uniq))

	// выводим результат
	fmt.Println("Уникальные слова:", keys)
}
