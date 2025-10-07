/* ВАРИАНТ №1 - решение задачи l1.23 с использованием пакета slices */

package main

import (
	"fmt"
	"slices"
)

func main() {

	idxToDel := 5 // индекс элемента, который надо удалить

	input := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15} // слайс из которого удаляем

	// выводим начальные показатели длины и ёмкости среза
	fmt.Println(input)
	fmt.Println("Начальная длина: len(input) =", len(input))
	fmt.Println("Начальная ёмкость: cap(input) =", cap(input))
	fmt.Println()

	//
	s1 := input[:idxToDel]
	s2 := input[idxToDel+1:]
	input = slices.Concat(s1, s2)

	fmt.Println(input)
	fmt.Println("Длина после slices.Concat: len(input) =", len(input))
	fmt.Println("Ёмкость после slices.Concat: cap(input) =", cap(input))

}
