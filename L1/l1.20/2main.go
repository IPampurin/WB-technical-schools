/* ВАРИАНТ №2 - решение задачи с использованием канала (так себе вариант) */

package main

import (
	"fmt"
	"sync"
)

// textRevers разворачивает строковую переменную в обратную сторону
func textRevers(s string) string {

	runes := []rune(s) // представляем входящую строку как слайс рун

	if len(runes) == 0 {
		return s
	}

	// отрезаем конечные пробелы, если они есть
	for runes[len(runes)-1] == ' ' {
		runes = runes[:len(runes)-1]
	}

	ch := make(chan []rune, len(runes)) // канал для передачи частей входящей строки

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		finish := len(runes) - 1 // индекс конца отрезка
		// идём с хвоста слайса рун
		for i := len(runes) - 1; i >= 0; i-- {
			// если встречаем пробел или доходим до начала текста,
			// отправляем пройденную часть в канал
			if runes[i] == ' ' {
				word := runes[i+1 : finish+1]
				finish = i - 1
				ch <- word
				ch <- []rune{' '}
			}
			if i == 0 {
				word := runes[i : finish+1]
				ch <- word
			}
		}
		close(ch)
	}()

	wg.Wait()

	// обнуляем входную строку
	runes = []rune{}

	// слушаем канал ch и перезаписываем входную строку
	for v := range ch {
		runes = append(runes, v...)
	}

	// преобразуем слайс в строковую переменную и возвращаем
	return string(runes)
}

func main() {

	input := "snow dog sun" // входящая строка

	input = textRevers(input) // меняем строку

	fmt.Println(input) // выводим результат
}
