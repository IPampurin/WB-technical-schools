/* ВАРИАНТ №1 - решение задачи l2.11 */

package main

import (
	"fmt"
	"slices"
	"strings"
)

// sortAnagram распределяет анаграммы по первому встреченному слову
func sortAnagram(words []string) map[string][]string {

	// мапа с анаграммами
	result := make(map[string][]string)

	// вспомогательная мапа для отслеживания
	// первого вхождения слова множества
	anaKeys := make(map[string]string)

	// проходим по словам поступившего слайса
	for _, word := range words {

		// приводим слово к нижнему регистру
		word = strings.ToLower(word)

		// разбиваем слово на руны
		runes := []rune(word)
		// сортируем руны
		slices.Sort(runes)
		// делаем из сортированных рун строку - ключ вспомогательной мапы
		key := string(runes)

		// проверяем есть ли такой ключ в anaKeys
		if _, ok := anaKeys[key]; !ok {
			// если слово из множества не встречали, то заносим
			// первое встреченное слово множества как значение
			anaKeys[key] = word
			// и добавляем в соответствующее множество встреченное слово
			result[anaKeys[key]] = append(result[anaKeys[key]], word)
		} else {
			// а если слова из множества уже встречались,
			// то добавляем в соответствующее множество встреченное слово
			result[anaKeys[key]] = append(result[anaKeys[key]], word)
		}

	}

	return result
}

func main() {

	input := []string{"пятак", "пятка", "тяпка", "листок", "слиток", "столик", "стол"}

	anagrams := sortAnagram(input)

	for key, val := range anagrams {
		if len(val) != 1 {
			fmt.Printf("%s: %v\n", key, val)
		}
	}
}
