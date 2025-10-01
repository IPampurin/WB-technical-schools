/* ВАРИАНТ №1 - решение задачи l1.16 */

package main

import (
	"fmt"
	"slices"
)

func main() {

	input1 := []int{1, 5, 1, 5, 6, 3, 2, 1, 7, 8, 9, 10, 2, 20, 45, 78, 51, 6, 2, 66, 89, 6, 3, 2, 1, 7, 8, 9, 10, 2, 20, 45, 78, 51, 6, 2, 66, 89}
	input2 := []int{1, 5, 6, 3, 2, 1, 5, 6, 3, 2, 1, 7, 8, 9, 10, 2, 20, 45, 78, 51, 6, 2, 66, 89, 1, 7, 8, 9, 10, 2, 20, 45, 78, 51, 6, 2, 66, 89}
	input3 := []int{1, 5, 6, 3, 2, 1, 5, 6, 3, 2, 1, 7, 8, 9, 10, 2, 20, 45, 78, 51, 6, 2, 66, 89, 1, 7, 8, 9, 10, 2, 20, 45, 78, 51, 6, 2, 66, 89}

	slices.Sort(input1)
	fmt.Println(input1)

	lowIndex2 := 0
	highIndex2 := len(input2) - 1

	input2 = quickSortHoar(input2, lowIndex2, highIndex2)
	fmt.Println(input2)

	lowIndex3 := 0
	highIndex3 := len(input3) - 1
	quickSortLomuto(input3, lowIndex3, highIndex3)
	fmt.Println(input3)
}

// схема разделения Хоара
func quickSortHoar(arr []int, low, high int) []int {
	if low >= high {
		return arr
	}

	left, right := low, high
	pivot := arr[(low+high)/2]

	for left <= right {
		for arr[right] > pivot {
			right--
		}
		for arr[left] < pivot {
			left++
		}
		if left <= right {
			arr[left], arr[right] = arr[right], arr[left]
			left++
			right--
		}
	}

	quickSortHoar(arr, low, right)
	quickSortHoar(arr, left, high)

	return arr
}

// схема разделения Ломуто
func partition(arr []int, low, high int) int {
	index := low - 1
	pivotElement := arr[high]
	for i := low; i < high; i++ {
		if arr[i] <= pivotElement {
			index += 1
			arr[index], arr[i] = arr[i], arr[index]
		}
	}
	arr[index+1], arr[high] = arr[high], arr[index+1]
	return index + 1
}

// quickSortLomuto Sorts the specified range within the array
func quickSortLomuto(arr []int, low, high int) {

	if len(arr) <= 1 {
		return
	}

	if low < high {
		pivot := partition(arr, low, high)
		quickSortLomuto(arr, low, pivot-1)
		quickSortLomuto(arr, pivot+1, high)
	}
}
