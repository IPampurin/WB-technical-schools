/* ВАРИАНТ №1 - решение задачи l2.11 (O(n × m log m))*/

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

	// вспомогательная мапа для отслеживания повторений
	unique := make(map[string]struct{})

	// проходим по словам поступившего слайса
	for _, word := range words {

		// приводим слово к нижнему регистру
		word = strings.ToLower(word)

		// пропускаем дубликаты
		if _, ok := unique[word]; ok {
			continue
		}

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
		}
		// и добавляем в соответствующее множество встреченное слово
		result[anaKeys[key]] = append(result[anaKeys[key]], word)
	}

	// проходим по мапе, сортируем множества и удаляем единичные множества
	for key, val := range result {
		if len(val) <= 1 {
			delete(result, key)
		} else {
			slices.Sort(val)
		}
	}

	return result
}

func main() {

	input := []string{"пятак", "пятка", "тяпка", "листок", "слиток", "столик", "стол"}

	anagrams := sortAnagram(input) // получаем мапу с множествами

	// итерируемся по мапе и выводим результат
	for key, val := range anagrams {
		fmt.Printf("%s: %v\n", key, val)
	}
}
