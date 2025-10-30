/* ВАРИАНТ №1 - решение задачи l2.12 Утилита grep */

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

// Config - конфигурация фильтрации
type Config struct {
	after      int    // количество символов после
	before     int    // количество символов до
	context    int    // количество символов вокруг
	count      bool   // надо только количество совпадений строк?
	ignoreCase bool   // надо игнорировать регистр?
	invert     bool   // надо инвертировать фильтр?
	fixed      bool   // надо точное совпадение?
	lineNumber bool   // надо пронумеровать совпадения?
	pattern    string // строка для фильтра
	filename   string // имя файла
}

// parseFlags парсит флаги строки запуска программы
func parseFlags() Config {

	config := Config{}
	flag.IntVar(&config.after, "A", 0, "Print N lines after match")
	flag.IntVar(&config.before, "B", 0, "Print N lines before match")
	flag.IntVar(&config.context, "C", 0, "Print N lines of context around match")
	flag.BoolVar(&config.count, "c", false, "Print only count of matching lines")
	flag.BoolVar(&config.ignoreCase, "i", false, "Ignore case")
	flag.BoolVar(&config.invert, "v", false, "Invert match")
	flag.BoolVar(&config.fixed, "F", false, "Treat pattern as fixed string")
	flag.BoolVar(&config.lineNumber, "n", false, "Print line numbers")

	flag.Parse() // парсим флаги из командной строки (Must be called after all flags are defined and before flags are accessed by the program)

	// возвращаем структуру конфигурации
	return config
}

// Line представляет строку с ввода
type Line struct {
	number  int    // номер строки
	content string // содержимое строки
}

func grep(config Config, input io.Reader) error {

	return nil
}

func main() {

	config := parseFlags() // парсим флаги запуска

	// получаем шаблон и имя файла
	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Используйте: grep [-флаги] шаблон [файл]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// шаблон поиска - первый аргумент
	config.pattern = args[0]

	var input io.Reader // объявляем ридер

	if len(args) > 1 {
		// если второй аргумент (файл) при запуске указан, то читаем из файла
		config.filename = args[1] // имя файла - второй аргумент

		file, err := os.Open(config.filename)
		if err != nil { // обрабатываем ошибку открытия файла
			fmt.Fprintf(os.Stderr, "ошибка открытия файла: %v\n", err)
			os.Exit(1)
		}
		defer file.Close() // обеспечиваем закрытие файла
		input = file

	} else {
		// а если при запуске программы имя файла не указано, то читаем из консоли
		input = os.Stdin
	}

	// выполняем фильтрацию, с учётом обработки ошибок
	if err := grep(config, input); err != nil {
		fmt.Fprintf(os.Stderr, "ошибка выполнения: %v\n", err)
		os.Exit(1)
	}
}
