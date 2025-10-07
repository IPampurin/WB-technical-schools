/* ВАРИАНТ №1 - решение задачи l1.23 с использованием встроеной функции copy */

package main

import (
	"fmt"
)

func main() {

	idxToDel := 1 // индекс элемента, который надо удалить

	// подопытный слайс
	input := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}

	// выводим начальные показатели длины и ёмкости среза
	fmt.Println(input)
	fmt.Printf("Начальная длина: len(input) = %d\n", len(input))
	fmt.Printf("Начальная ёмкость: cap(input) = %d\n", cap(input))
	fmt.Println()

	// сдвигаем хвост слайса влево, то есть
	// копируем элементы из input[idxToDel+1:] в input[idxToDel:],
	// перезаписывая удаляемый элемент и повторяя последний элемент
	copy(input[idxToDel:], input[idxToDel+1:])

	// удаляем крайний правый элемент, то есть
	// создаём новый слайс без последнего элемента
	input = input[:len(input)-1]

	// выводим конечные показатели длины и ёмкости среза
	fmt.Println(input)
	fmt.Println("Длина после удаления: len(input) =", len(input))
	fmt.Println("Ёмкость после удаления: cap(input) =", cap(input))

}
