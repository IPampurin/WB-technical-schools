/* ВАРИАНТ №1 - решение задачи l2.9, функция unpackingString() */

package main

import (
	"fmt"
	"strconv"
	"unicode"
)

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

	// если в строке только цифры
	if _, err := strconv.Atoi(string(runes[0])); err == nil {
		return "", fmt.Errorf("некорректная строка: цифра в начале")
	}

	// если цифра в начале строки, выдаём ошибку (под этот случай подпадает и строка из цифр)
	if len(runes) == 1 && unicode.IsDigit(runes[0]) {
		return "", fmt.Errorf("некорректная строка: цифра в начале")
	}

	result := make([]rune, 0) // место под распаковку
	escapeFlag := false       // флаг встречи с '\'

	// итерируемся по runes
	for i := 0; i < len(runes); i++ {

		if runes[i] == '\\' {
			escapeFlag = true
		}
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
		"a10b2",     // "aaaaaaaaaabb"
		"\\\\a",     // "\a"
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
