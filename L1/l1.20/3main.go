/* ВАРИАНТ №3 - решение задачи с использованием container/list */

package main

import (
	"container/list"
	"fmt"
)

// textRevers разворачивает строковую переменную в обратную сторону
func textRevers(s string) string {

	runes := []rune(s) // представляем входящую строку как слайс рун

	// создаём новую очередь
	l := list.New()

	// идём по runes и если встречаем пробел или конец данных, ставим отрезок в начало очереди
	start := 0 // начало отрезка для записи в очередь
	for i := 0; i < len(runes); i++ {
		if runes[i] == ' ' || i == len(runes)-1 {
			end := i // конец отрезка для записи в очередь
			if i == len(runes)-1 && runes[i] != ' ' {
				end++ // включаем последний символ
			}
			word := runes[start:end]
			l.PushFront(word) // добавляем word в начало очереди
			start = i + 1     // смещаем индекс начала отрезка
		}
	}
	// обнуляем runes
	runes = []rune{}

	// проходим по очереди и пишем элементы очереди в runes с учётом пробелов
	for e := l.Front(); e != nil; e = e.Next() {
		if word, ok := e.Value.([]rune); ok { // приводим тип к []rune из any и проверяем корректность приведения
			if len(runes) > 0 { // добавляем пробел перед словом, если это не первое слово
				runes = append(runes, ' ')
			}
			runes = append(runes, word...)
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
