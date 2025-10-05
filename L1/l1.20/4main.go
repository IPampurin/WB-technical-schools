/* ВАРИАНТ №4 - решение с полным и частичным реверсом (лучший вариант по памяти и производительности) */

package main

import (
	"fmt"
)

// revers разворачивает поступивший слайс рун
func revers(word []rune) {

	for i, j := 0, len(word)-1; i < j; i, j = i+1, j-1 {
		word[i], word[j] = word[j], word[i]
	}

}

// textRevers разворачивает строковую переменную в обратную сторону
func textRevers(s string) string {

	runes := []rune(s) // представляем входящую строку как слайс рун

	revers(runes) // сначала разворачиваем весь поступивший текст

	// идём по развёрнутому тексту и возвращаем отдельные слова в нормальное положение
	start := 0
	for i := 0; i < len(runes); i++ {
		if runes[i] == ' ' { // обрабатываем первые слова
			revers(runes[start:i])
			start = i + 1
		}
		if i == len(runes)-1 { // обрабатываем последнее слово
			revers(runes[start : i+1])
		}
	}

	// преобразуем слайс в строковую переменную и возвращаем
	return string(runes)
}

func main() {

	input := "snow dog sun" // входящая строка

	input = textRevers(input) // меняем строку

	fmt.Println(input) // выводим результат
}
