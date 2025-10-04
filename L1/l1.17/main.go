/* ВАРИАНТ №1 - решение задачи реализации бинарного поиска (l1.17) итеративно и рекурсивно */

/* Примечание: итеративная версия находит первое вхождение элемента (самый левый индекс),
   а рекурсивная версия находит произвольное вхождение (любой подходящий индекс). */

package main

import "fmt"

// iterationBinSearch - реализация алгортма бинарного поиска итеративно
func iterationBinSearch(findNumber int, arr []int) int {

	// если массив пустой, сразу возвращаем -1
	if len(arr) == 0 {
		return -1
	}

	// определяем первоначальные границы поиска
	left, right := 0, len(arr)-1

	// пока границы поиска захватывают хотя бы один элемент выполняем проверку
	for left <= right {
		mid := left + (right-left)/2 // защита от переполнения на случай ну очень большого массива
		if findNumber < arr[mid] {
			right = mid - 1 // если искомый элемент левее, смещаем правую границу
		} else if arr[mid] < findNumber {
			left = mid + 1 // если искомый элемент правее, смещаем левую границу
		} else {
			return mid // если нашли совпадение, возвращаем индекс совпадения
		}
	}

	// если совпадений не нашлось, возвращаем -1
	return -1
}

// recursionBinSearch - реализация алгортма бинарного поиска рекурсивно
func recursionBinSearch(findNumber int, arr []int) int {

	// если массив пустой, сразу возвращаем -1
	if len(arr) == 0 {
		return -1
	}

	mid := len(arr) / 2 // индекс среднего элемента

	if findNumber < arr[mid] {
		return recursionBinSearch(findNumber, arr[:mid]) // если искомый элемент левее, ищем в левой половине
	} else if arr[mid] < findNumber {
		searchIndex := recursionBinSearch(findNumber, arr[mid+1:]) // если искомый элемент правее, ищем в правой половине
		if searchIndex == -1 {
			return -1
		}
		// корректируем индекс, так как индексы переданной правой половины массива сдвинуты относительно исходного массива
		return mid + 1 + searchIndex
	} else {
		return mid // если нашли совпадение, возвращаем индекс совпадения
	}
}

func main() {

	findNumber := 66 // число, которое будем искать в массиве входных данных

	// input1 отсортированные входные данные для итеративной версии
	input1 := []int{1, 1, 2, 3, 3, 4, 5, 6, 6, 6, 7, 8, 9, 10, 15, 20, 30, 35, 35, 45, 45, 51, 52, 66, 66, 67, 78, 87, 89, 96}
	// input2 отсортированные входные данные для рекурсивной версии
	input2 := []int{1, 1, 2, 3, 3, 4, 5, 6, 6, 6, 7, 8, 9, 10, 15, 20, 30, 35, 35, 45, 45, 51, 52, 66, 66, 67, 78, 87, 89, 96}

	// применяем итеративный подход к поиску
	searchIndex := iterationBinSearch(findNumber, input1)
	if searchIndex == -1 {
		fmt.Printf("Итеративный подход: число %d не найдено в массиве.\n", findNumber)
	} else {
		fmt.Printf("Итеративный подход: число %d найдено в массиве по индексу %d.\n", findNumber, searchIndex)
	}

	// применяем рекурсивный подход к поиску
	searchIndex = recursionBinSearch(findNumber, input2)
	if searchIndex == -1 {
		fmt.Printf("Рекурсивный подход: число %d не найдено в массиве.\n", findNumber)
	} else {
		fmt.Printf("Рекурсивный подход: число %d найдено в массиве по индексу %d.\n", findNumber, searchIndex)
	}
}
