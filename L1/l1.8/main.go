/* ВАРИАНТ №1 - установка i-го бита числа в 1 или 0 */

package main

import "fmt"

// SetBit устанавливает i-й бит числа value в 1 или 0
// i отсчитываем от 1 (младший бит справа)
func SetBit(value int64, i int, bitValue int) int64 {

	if bitValue == 1 {
		// установка бита в 1 с помощью OR
		// с учётом условия, что младший бит не нулевой, а первый
		return value | (1 << (i - 1))
	} else {
		// установка бита в 0 с помощью AND NOT
		// с учётом условия, что младший бит не нулевой, а первый
		return value &^ (1 << (i - 1))
	}
}

func main() {

	var number int64 = 5    // входящее число в десятичном формате
	var bitPosition int = 1 // нумеруем биты с номера один (по условию задачи), справа налево
	var newBitValue int = 0 // или 0 или 1

	// валидируем bitPosition
	if uint(bitPosition-1) > 63 {
		// (bitPosition - 1) преобразуем в uint (проверяем на отрицательность)
		// если результат > 63, значит исходный bitPosition > 64
		fmt.Println("bitPosition должен быть в пределах от 1 до 64")
		return
	}
	// побитовая проверка newBitValue
	if newBitValue&^1 != 0 { // newBitValue&^1 != 0 значит не 0 и не 1
		fmt.Println("newBitValue должен быть или 0, или 1")
		return
	}

	fmt.Printf("\nИсходное число:\n	- в десятичной форме %d\n	- в двоичном формате %064b\n", number, number)
	fmt.Printf("Позиция бита, который меняем: %d\n", bitPosition)
	fmt.Printf("Новое значение бита: %d\n", newBitValue)

	result := SetBit(number, bitPosition, newBitValue)

	fmt.Printf("\nНовое число:\n	- в десятичной форме %d\n	- в двоичном формате %064b\n", result, result)

}
