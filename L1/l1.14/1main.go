/* ВАРИАНТ №1 - решение с применением switch v.(type) */

package main

import "fmt"

// parseInput сообщает о том, какого типа поступивший объект
func parseInput(input any) string {

	if input == nil {
		return "Поступвшая переменная имеет значение nil.\n"
	}

	mes := ""

	switch v := input.(type) {
	case int:
		mes = fmt.Sprintf("Поступившая переменная '%d' имеет тип int\n", v)
	case string:
		mes = fmt.Sprintf("Поступившая переменная '%s' имеет тип string\n", v)
	case bool:
		mes = fmt.Sprintf("Поступившая переменная '%t' имеет тип bool\n", v)
	case chan int:
		mes = fmt.Sprintf("Поступившая переменная '%v' имеет тип chan int\n", v)
	default:
		return fmt.Sprintf("неизвестный тип переменной: %T\n", v)
	}

	return mes
}

func main() {

	inputs := []any{52, "string", true, make(chan int), nil, 3.14}

	for _, input := range inputs {
		fmt.Println(parseInput(input))
	}
}
