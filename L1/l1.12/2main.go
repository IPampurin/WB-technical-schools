/* ВАРИАНТ №2 - нахождение уникальных слов на конвейере */

// Примечание: такая архитектура избыточна для данной задачи.

package main

import (
	"fmt"
	"slices"
)

// prod пишет слова по очереди в канал для передачи на обработку
func prod(ch chan<- string, words []string) {

	for _, v := range words {
		ch <- v
	}
	close(ch)
}

// processor проверяет было ли уже такое слово и отправляет дальше только уникальные слова
func processor(chIn <-chan string, chOut chan<- string) {

	// мапа для уникальных строк как ключей
	// структура-значение чтобы не занимать память
	uniq := make(map[string]struct{})

	// читаем chIn, записываем слова в мапу и отправляем дальше уникальные слова
	for v := range chIn {
		if _, ok := uniq[v]; !ok {
			uniq[v] = struct{}{}
			chOut <- v
		}
	}
	close(chOut)
}

func main() {

	input := []string{"cat", "cat", "dog", "cat", "tree"} // входные строки

	in := make(chan string)  // канал передачи слов в работу
	out := make(chan string) // канал получения обработанных слов

	go prod(in, input)    // отправляем слова в работу
	go processor(in, out) // находим уникальные слова

	result := make([]string, 0) // слайс для найденных уникальных слов
	// читаем out до его закрытия и пишем в result
	for v := range out {
		result = append(result, v)
	}

	slices.Sort(result) // отсортируем для красоты

	fmt.Println("Уникальные слова:", result) // выводим результат
}
