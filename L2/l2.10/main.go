/* ВАРИАНТ №1 - решение задачи l2.10 */

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
)

// Config - конфигурация сортировки
type Config struct {
	keyColumn            int    // флаг сортировки по столбцам
	numeric              bool   // флаг числовой сортировки
	reverse              bool   // флаг сортировки в обратном порядке
	unique               bool   // флаг выдачи без повторов
	month                bool   // сортировка по месяцам
	ignoreTrailingBlanks bool   // флаг игнора хвостовых пробелов
	checkSorted          bool   // флаг проверки на отсортированность
	humanNumeric         bool   // флаг сортировки по размерам
	columnSeparator      string // разделитель по умолчанию (табуляция)
}

// parseFlags парсит флаги строки запуска программы
func parseFlags() Config {

	config := Config{}
	flag.IntVar(&config.keyColumn, "k", 0, "sort by column N")
	flag.BoolVar(&config.numeric, "n", false, "sort numerically")
	flag.BoolVar(&config.reverse, "r", false, "sort in reverse order")
	flag.BoolVar(&config.unique, "u", false, "output only unique lines")
	flag.BoolVar(&config.month, "M", false, "sort by month")
	flag.BoolVar(&config.ignoreTrailingBlanks, "b", false, "ignore trailing blanks")
	flag.BoolVar(&config.checkSorted, "c", false, "check if sorted")
	flag.BoolVar(&config.humanNumeric, "h", false, "sort by human-readable sizes")

	flag.Parse() // парсим флаги из командной строки (Must be called after all flags are defined and before flags are accessed by the program)

	config.columnSeparator = "\t" // разделитель по умолчанию (табуляция)

	// возвращаем структуру
	return config
}

// readLines считывает ввод по строкам
func readLines(r io.Reader) []string {

	scanner := bufio.NewScanner(r)
	const maxCapacity = 1024 * 1024 * 1024 // 1GB
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "ошибка считывания: %v\n", err)
		os.Exit(1)
	}
	return lines
}

func main() {

	config := parseFlags() // парсим флаги запуска

	var input io.Reader // объявляем ридер

	if fileName := flag.Arg(0); fileName != "" {
		// если при запуске программы указано имя файла, то читаем из него
		file, err := os.Open(fileName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ошибка открытия файла: %v\n", err)
			os.Exit(1)
		}
		defer file.Close() // обеспечиваем закрытие файла
		input = file
	} else {
		// если при запуске программы имя файла не указано, то читаем из консоли
		input = os.Stdin
	}

	lines := readLines(input)

}
