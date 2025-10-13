/* ВАРИАНТ №1 - решение задачи l2.9, функция unpackingString() */

package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

func unpackingString(str string) (string, error) {
	/*
		if len([]rune(str)) == 0 {
			return "", nil
		}
		if len([]rune(str)) == 1 && str == "\\" {
			return "", fmt.Errorf("некорректная строка, т.к. в строке только знак экранирования\n")
		}
		if _, err := strconv.Atoi(str); err == nil {
			return "", fmt.Errorf("некорректная строка, т.к. в строке только цифры\n")
		}
	*/
	result := make([]rune, 0)
	var prevSymbol rune

	for _, v := range str {

		if unicode.IsLetter(v) {
			result = append(result, v)
		}
		if unicode.IsDigit(v) {
			n, err := strconv.Atoi(string(v))
			if err != nil {
				return "", fmt.Errorf("ошибка парсинга числа 'v': %w ", err)
			}
			repeatSymbol := strings.Repeat(string(prevSymbol), n)
			result = append(result, []rune(repeatSymbol)...)
		}
		prevSymbol = v
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
