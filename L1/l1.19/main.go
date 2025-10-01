/* ВАРИАНТ №1 - решение задачи l1.19 с использованием []rune */

package main

import (
	"fmt"
)

// revers разворачивает строковую переменную в обратную сторону
func revers(s string) string {

	runes := []rune(s) // представляем входящую строку как слайс рун

	// проходим до половины слайса и меняем элемент слайса с противоположным
	for i := 0; i < len(runes)/2; i++ {
		runes[i], runes[len(runes)-1-i] = runes[len(runes)-1-i], runes[i]
	}

	// преобразуем слайс в строковую переменную и возвращаем
	return string(runes)
}

func main() {

	input := "главрыба" // входящая строка

	input = revers(input) // меняем строку

	fmt.Println(input) // выводим результат
}
