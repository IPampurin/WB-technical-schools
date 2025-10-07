/* ВАРИАНТ №2, 3 - решение задачи l1.23 с использованием пакета slices */

package main

import (
	"fmt"
	"slices"
)

func main() {

	idxToDel := 0 // индекс элемента, который надо удалить (проверку на выход заграницы массива опустим)

	// подопытный слайс
	input := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}

	// выводим начальные показатели длины и ёмкости среза
	fmt.Println(input)
	fmt.Printf("Начальная длина: len(input) = %d\n", len(input))
	fmt.Printf("Начальная ёмкость: cap(input) = %d\n", cap(input))
	fmt.Println()

	// вариант с использованием slices.Concat (проигрывает в выделении памяти)
	s1 := input[:idxToDel]        // создаём новый слайс от 0 до idxToDel
	s2 := input[idxToDel+1:]      // создаём новый слайс от idxToDel+1 до len(input)
	input = slices.Concat(s1, s2) // перезаписываем input

	// выводим конечные показатели длины и ёмкости среза
	fmt.Println(input)
	fmt.Println("Длина после slices.Concat: len(input) =", len(input))
	fmt.Println("Ёмкость после slices.Concat: cap(input) =", cap(input))
	fmt.Println()

	// вариант с использованием slices.Delete (самый читабельный, с проверками и очисткой под капотом)
	input = slices.Delete(input, idxToDel, idxToDel+1)

	// выводим конечные показатели длины и ёмкости среза
	fmt.Println(input)
	fmt.Println("Длина после slices.Delete: len(input) =", len(input))
	fmt.Println("Ёмкость после slices.Delete: cap(input) =", cap(input))
}
