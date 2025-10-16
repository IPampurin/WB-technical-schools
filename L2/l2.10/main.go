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
	keyColumn            int //
	numeric              bool
	reverse              bool
	unique               bool
	month                bool
	ignoreTrailingBlanks bool
	checkSorted          bool
	humanNumeric         bool
	columnSeparator      string
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
	flag.Parse()

	config.columnSeparator = "\t" // по умолчанию табуляция
	return config
}

// readLines считывает
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

	config := parseFlags()

	var input io.Reader
	if filename := flag.Arg(0); filename != "" {
		file, err := os.Open(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ошибка открытия файла: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		input = file
	} else {
		input = os.Stdin
	}

	lines := readLines(input)

}
