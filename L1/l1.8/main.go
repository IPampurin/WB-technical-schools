/* ВАРИАНТ №1 - установка i-го бита числа в 1 или 0 */

package main

import "fmt"

// SetBit устанавливает i-й бит числа value в 1 или 0
// i отсчитываем от 1 (младший бит справа)
func SetBit(value int64, i int, bitValue int) int64 {

	if bitValue == 1 {
		// установка бита в 1 с помощью OR
		return value | (1 << (i - 1))
	} else {
		// установка бита в 0 с помощью AND NOT
		return value &^ (1 << (i - 1))
	}
}

func main() {

	var number int64 = 5    // входящее число в десятичном формате
	var bitPosition int = 4 // нумеруем биты с номера один, справа налево
	var newBitValue int = 1 // или 0 или 1

	fmt.Printf("\nИсходное число:\n	- в десятичной форме %d\n	- в двоичном формате %064b\n", number, number)
	fmt.Printf("Позиция бита, который меняем: %d\n", bitPosition)
	fmt.Printf("Новое значение бита: %d\n", newBitValue)

	result := SetBit(number, bitPosition, newBitValue)

	fmt.Printf("\nНовое число:\n	- в десятичной форме %d\n	- в двоичном формате %064b\n", result, result)

}
