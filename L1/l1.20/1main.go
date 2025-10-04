/* ВАРИАНТ №1 - решение задачи с использованием пакета strings */

package main

import (
	"fmt"
	"strings"
)

// textRevers разворачивает строковую переменную в обратную сторону
func textRevers(s string) string {

	// разделяем строку по пробелам сколько бы их ни было
	words := strings.Fields(s)

	// зеркалим слайс входящих слов
	for i, j := 0, len(words)-1; i < j; i, j = i+1, j-1 {
		words[i], words[j] = words[j], words[i]
	}

	// объединяем отзеркаленный слайс в строку с пробелами и возвращаем
	return strings.Join(words, " ")
}

func main() {

	input := "snow dog sun" // входящая строка

	input = textRevers(input) // меняем строку

	fmt.Println(input) // выводим результат
}
