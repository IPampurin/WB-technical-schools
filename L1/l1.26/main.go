/* ВАРИАНТ №1 - решение задачи l1.26 */

package main

import (
	"fmt"
	"strings"
)

// isUniqString проверяет есть ли повторяющиеся символы в строке вне зависимости от регистра
func isUniqString(str string) bool {

	// приводим поступившую строку к нижнему регистру
	strLower := strings.ToLower(str)

	// uniq мапа, в которой ключи - уникальные символы строки,
	// а значения - пустые структуры, чтобы не занимать место
	uniq := make(map[rune]struct{})

	// проходим по строке и проверяем в мапе наличие символа в ключах uniq
	for _, v := range strLower {
		if _, ok := uniq[v]; ok {
			return false
		}
		uniq[v] = struct{}{}
	}

	return true
}

func main() {

	// придумаем набор входных данных (включая кириллическую х)
	input := []string{"abcd", "abCdefAaf", "aabcd", "1bcd", "1bcd1", "1bCd", "ab_cd", "aA", "ХX"}

	// проходим по поступившим строкам и проверяем на уникальность составляющих символов
	for _, v := range input {
		if ok := isUniqString(v); ok {
			fmt.Printf("Строка %20s  -  %t\n", v, ok)
		} else {
			fmt.Printf("Строка %20s  -  %t\n", v, ok)
		}
	}
}
