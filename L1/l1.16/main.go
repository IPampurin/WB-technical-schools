/* ВАРИАНТ №1 - решение задачи l1.16 с использованием четырёх вариантов быстрой сортировки */

package main

import (
	"fmt"
	"slices"
)

// quickSortFirst схема с первым элементом в качестве опорного
// в худшем случае сложность n^2 (при отсортированном массиве)
func quickSortFirst(arr []int, low, high int) {

	// если массив пустой или с одним элементом, он уже отсортирован
	if len(arr) <= 1 {
		return
	}

	// условие выхода из рекурсии
	if low < high {
		index := low
		pivotElement := arr[low] // за опорный элемент принимаем первый

		for i := low + 1; i <= high; i++ {
			// если встречаем элемент <= опроного, увеличиваем границу
			// и перемещаем этот элемент в левую часть
			if arr[i] <= pivotElement {
				index++
				arr[index], arr[i] = arr[i], arr[index]
			}
		}

		// ставим опорный элемент на окончательную позицию
		arr[low], arr[index] = arr[index], arr[low]

		pivot := index // новый индекс опорного элемента

		quickSortFirst(arr, low, pivot-1)  // рекурсивно вызываем функцию для левой от опорного половины массива
		quickSortFirst(arr, pivot+1, high) // рекурсивно вызываем функцию для правой от опорного половины массива
	}
}

// quickSortHoar схема разделения Хоара
// средняя сложность n*log n, худший случай n^2
func quickSortHoar(arr []int, low, high int) {

	// если подмассив пустой или содержит один элемент
	if low >= high {
		return
	}

	left, right := low, high          // указатели для поиска элементов слева и справа соответственно
	pivotElement := arr[(low+high)/2] // за опорный элемент принимаем средний

	// пока указатели движутся навстречу
	for left <= right {
		// находим крайний правый элемент <= опорному
		for arr[right] > pivotElement {
			right--
		}
		// находим крайний левй элемент >= опорному
		for arr[left] < pivotElement {
			left++
		}

		// если указатели не пересеклись
		if left <= right {
			arr[left], arr[right] = arr[right], arr[left] // меняем найденные элементы местами
			left++
			right--
		}
	}

	quickSortHoar(arr, low, right) // рекурсивно вызываем функцию для левой от опорного половины массива
	quickSortHoar(arr, left, high) // рекурсивно вызываем функцию для правой от опорного половины массива
}

// quickSortLomuto схема разделения Ломуто
// в худшем случае сложность n^2 (при отсортированном массиве)
func quickSortLomuto(arr []int, low, high int) {

	// если массив пустой или с одним элементом, он уже отсортирован
	if len(arr) <= 1 {
		return
	}

	// условие выхода из рекурсии
	if low < high {
		index := low - 1
		pivotElement := arr[high] // за опорный элемент принимаем последний

		for i := low; i < high; i++ {
			// если встречаем элемент <= опроного, увеличиваем границу
			// и перемещаем этот элемент в левую часть
			if arr[i] <= pivotElement {
				index += 1
				arr[index], arr[i] = arr[i], arr[index]
			}
		}

		// ставим опорный элемент на окончательную позицию
		arr[index+1], arr[high] = arr[high], arr[index+1]

		pivot := index + 1 // новый индекс опорного элемента

		quickSortLomuto(arr, low, pivot-1)  // рекурсивно вызываем функцию для левой от опорного половины массива
		quickSortLomuto(arr, pivot+1, high) // рекурсивно вызываем функцию для правой от опорного половины массива
	}
}

func main() {

	input1 := []int{1, 5, 1, 5, 6, 3, 2, 1, 7, 8, 9, 10, 2, 20, 45, 78, 51, 6, 2, 66, 89, 6, 3, 2, 1, 7, 8, 9, 10, 2, 20, 45, 78, 51, 6, 2, 66, 89}
	input2 := []int{1, 5, 6, 3, 2, 1, 5, 6, 3, 2, 1, 7, 8, 9, 10, 2, 20, 45, 78, 51, 6, 2, 66, 89, 1, 7, 8, 9, 10, 2, 20, 45, 78, 51, 6, 2, 66, 89}
	input3 := []int{1, 5, 6, 3, 2, 1, 5, 6, 3, 2, 1, 7, 8, 9, 10, 2, 20, 45, 78, 51, 6, 2, 66, 89, 1, 7, 8, 9, 10, 2, 20, 45, 78, 51, 6, 2, 66, 89}
	input4 := []int{1, 5, 6, 3, 2, 1, 5, 6, 3, 2, 1, 7, 8, 9, 10, 2, 20, 45, 78, 51, 6, 2, 66, 89, 1, 7, 8, 9, 10, 2, 20, 45, 78, 51, 6, 2, 66, 89}

	slices.Sort(input1) // самая быстрая оптимизированная сортировка всех времён и народов
	fmt.Println(input1)

	lowIndex2 := 0
	highIndex2 := len(input2) - 1
	quickSortHoar(input2, lowIndex2, highIndex2)
	fmt.Println(input2)

	lowIndex3 := 0
	highIndex3 := len(input3) - 1
	quickSortLomuto(input3, lowIndex3, highIndex3)
	fmt.Println(input3)

	lowIndex4 := 0
	highIndex4 := len(input4) - 1
	quickSortLomuto(input4, lowIndex4, highIndex4)
	fmt.Println(input4)
}
