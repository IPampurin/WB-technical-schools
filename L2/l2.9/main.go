/* ВАРИАНТ №1 - решение задачи l2.9, функция unpackingString() */

package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// unpackingString распаковывает строку с учётом escape-последовательностей
func unpackingString(str string) (string, error) {

	// если строка пустая, сразу возвращаем пустую чтроку
	if len(str) == 0 {
		return "", nil
	}
	// если в строке только знак экранирования, то это ошибка
	if len(str) == 1 && str == "\\" {
		return "", fmt.Errorf("некорректная строка: в строке только знак экранирования")
	}

	runes := []rune(str) // представляем вход как []rune

	// если цифра в начале строки, выдаём ошибку (под этот случай подпадает и строка из цифр)
	if unicode.IsDigit(runes[0]) {
		return "", fmt.Errorf("некорректная строка: цифра в начале")
	}

	result := make([]rune, 0) // место под распаковку
	escapeFlag := false       // флаг встречи с '\'

	// итерируемся по runes
	for i := 0; i < len(runes); i++ {

		// после экранировки добавляем символ как есть, переключаем флаг и идём за новым символом
		if escapeFlag {
			result = append(result, runes[i])
			escapeFlag = false
			continue
		}

		// если встречаем '\', переключаем флаг и идём за новым символом
		if runes[i] == '\\' {
			escapeFlag = true
			continue
		}

		// если экранировки нет и
		// если встречаем число, надо повторить предыдущий символ n-1 раз
		if unicode.IsDigit(runes[i]) {
			// находим конец числа
			numStart := i
			numEnd := i
			for numEnd < len(runes) && unicode.IsDigit(runes[numEnd]) {
				numEnd++
			}
			// извлекаем число
			countInRunes := string(runes[numStart:numEnd])
			n, err := strconv.Atoi(countInRunes)
			if err != nil {
				return "", fmt.Errorf("ошибка парсинга числа '%s': %w", countInRunes, err)
			}
			if n == 0 {
				return "", fmt.Errorf("некорректная строка: 0 в количестве повторений")
			}
			symbolRepeat := strings.Repeat(string(result[len(result)-1]), n-1)
			result = append(result, []rune(symbolRepeat)...)

			// перемещаем индекс за число
			i = numEnd - 1
		} else {
			// а если встретили не цифру, то просто добавляем в result
			result = append(result, runes[i])
		}
	}

	// если по окончанию итерации у нас escapeFlag == true, значит '\' в конце строки
	if escapeFlag {
		return "", fmt.Errorf("некорректная строка: escape-символ в конце")
	}

	return string(result), nil
}

func main() {

	input := []string{
		"a4bc2d5e",  // "aaaabccddddde"
		"abcd",      // "abcd"
		"45",        // "" + err (некорректная строка, т.к. в строке только цифры — функция должна вернуть ошибку)
		"",          // "" (пустая строка -> пустая строка)
		"qwe\\4\\5", // "qwe45" (4 и 5 не трактуются как числа, т.к. экранированы)
		"qwe\\45",   // "qwe44444" (\4 экранирует 4, поэтому распаковывается только 5)
		"\\45",      // "44444"
		"\\",        // "" + err (некорректная строка, т.к. в строке только знак экранирования)
		"abc\\",     // "" + err (некорректная строка, т.к. знак экранирования в конце строки)
		"\\\\a",     // "\a"
		"\\510",     // "5555555555"
		"a0",        // "" + err (некорректная строка, т.к. 0 в количестве повторений)
	}

	for _, v := range input {
		resUnpack, err := unpackingString(v)
		if err != nil {
			fmt.Printf("Строка: %10s,  ошибка: %v\n", v, err)
			continue
		}
		fmt.Printf("Строка: %10s,  распаковка: %s\n", v, resUnpack)
	}
}
