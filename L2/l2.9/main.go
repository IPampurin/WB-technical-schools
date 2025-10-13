/* ВАРИАНТ №1 - решение задачи l2.9, функция unpackingString() */

package main

import "fmt"

func UnpackingString(str string) (string, error) {

	result := ""

	return result, nil
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
		"\\",        // ""
	}

	for _, v := range input {
		resUnpack, err := UnpackingString(v)
		if err != nil {
			fmt.Printf("Строка: %10s,  ошибка: %v\n", v, err)
			continue
		}
		fmt.Printf("Строка: %10s,  распаковка: %s\n", v, resUnpack)
	}
}

/*
	{"a4bc2d5e", "aaaabccddddde", nil},
	{"abcd", "abcd", nil},
	{"45", "", err},
	{"", "", nil},
	{"qwe\\4\\5", "qwe45", nil},
	{"qwe\\45", "qwe44444", nil},
	{"\\", "", nil},
*/
