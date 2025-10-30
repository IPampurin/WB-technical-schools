/* ВАРИАНТ №1 - решение задачи l2.12 Утилита grep */

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
)

// Config - конфигурация фильтрации
type Config struct {
	after      int    // количество строк после (флаг -A)
	before     int    // количество строк до (флаг -B)
	context    int    // количество строк вокруг (флаг -C)
	count      bool   // надо только количество совпадений строк? (флаг -c)
	ignoreCase bool   // надо игнорировать регистр? (флаг -i)
	invert     bool   // надо инвертировать фильтр? (флаг -v)
	fixed      bool   // надо точное совпадение? (флаг -F)
	lineNumber bool   // надо пронумеровать совпадения? (флаг -n)
	pattern    string // шаблон для фильтра (регулярное выражение или фиксированная строка)
	filename   string // имя файла (если не указан - читаем из Stdin)
}

// parseFlags парсит флаги строки запуска программы
func parseFlags() Config {

	config := Config{}

	// устанавливаем сообщение об использовании на случай ошибки в написании флагов при запуске программы
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Используйте: grep [-флаги] шаблон [файл]\n")
		fmt.Fprintf(os.Stderr, "Флаги:\n")
		flag.PrintDefaults()
	}

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
	number  int    // номер строки в исходном потоке (начиная с 1)
	content string // содержимое строки
}

func grep(config Config, input io.Reader) (*[]Line, int, error) {

	scanner := bufio.NewScanner(input)

	var lines []Line // слайс для хранения всех строк с их номерами
	lineNum := 0     // счётчик строк для нумерации

	// считываем строки пока они есть, прибавляем счётчик
	// нумерации и добавляем информацию в слайс
	for scanner.Scan() {
		lineNum++
		lines = append(lines, Line{
			number:  lineNum,
			content: scanner.Text(),
		})
	}

	// обрабатывем ошибку сканера (файл кирдык)
	if err := scanner.Err(); err != nil {
		return nil, 0, fmt.Errorf("ошибка чтения входных данных: %w", err)
	}

	pattern := config.pattern // шаблон для поиска с учётом флагов

	// добавляем модификатор игнора регистра к регулярному выражению
	if config.ignoreCase {
		pattern = "(?i)" + pattern
	}

	// экранируем специальные символы если шаблон должен трактоваться как фиксированная строка
	if config.fixed {
		pattern = regexp.QuoteMeta(pattern)
	}

	// компилируем регулярное выражение
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, 0, fmt.Errorf("неверный шаблон: %w", err)
	}

	// определяем размер контекста для вывода
	before := config.before
	after := config.after
	// если задан общий контекст, он переопределяет отдельные флаги до/после
	if config.context > 0 {
		before = config.context
		after = config.context
	}

	// проверяем каждую строку на соответствие шаблону
	matches := make([]bool, len(lines)) // слайс флагов совпадений для каждой строки
	for i, line := range lines {
		matched := re.MatchString(line.content) // проверяем совпадение с регулярным выражением

		// инвертируем результат если установлен флаг -v
		if config.invert {
			matched = !matched
		}

		matches[i] = matched
	}

	// если необходимо только количество совпадений,
	// возвращаем это количество
	if config.count {
		count := 0

		for _, matched := range matches {
			if matched {
				count++
			}
		}

		return nil, count, nil
	}

	// используем множество для хранения индексов строк к выводу
	// это, вроде, решает проблему дублирования и пропусков при перекрывающемся контексте
	outputIndexes := make(map[int]bool)

	// проходим по всем строкам и для каждой найденной добавляем её и контекст вокруг
	for i, matched := range matches {

		if matched {
			// добавляем строки контекста ДО совпадения
			start := i - before
			if start < 0 {
				start = 0
			}
			for j := start; j < i; j++ {
				outputIndexes[j] = true
			}

			// добавляем саму строку с совпадением
			outputIndexes[i] = true

			// добавляем строки контекста ПОСЛЕ совпадения
			end := i + after
			if end >= len(lines) {
				end = len(lines) - 1
			}
			for j := i + 1; j <= end; j++ {
				outputIndexes[j] = true
			}
		}
	}

	// преобразуем множество индексов в отсортированный слайс
	// это гарантирует, что строки будут выведены в правильном порядке
	var sortedIndexes []int
	for idx := range outputIndexes {
		sortedIndexes = append(sortedIndexes, idx)
	}
	sort.Ints(sortedIndexes)

	// собираем строки для вывода в правильном порядке
	var outputLines []Line
	for _, idx := range sortedIndexes {
		outputLines = append(outputLines, lines[idx])
	}

	return &outputLines, 0, nil
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
	if outputLines, count, err := grep(config, input); err != nil {
		fmt.Fprintf(os.Stderr, "ошибка выполнения: %v\n", err)
		os.Exit(1)

	} else if count != 0 {
		// выводим количество в консоль
		fmt.Printf("%d\n", count)

	} else {
		// выводим результаты в консоль
		for _, line := range *outputLines {
			if config.lineNumber {
				// вывод с номером строки если установлен флаг -n
				fmt.Printf("%d:%s\n", line.number, line.content)
			} else {
				// вывод только содержимого строки
				fmt.Printf("%s\n", line.content)
			}
		}
	}
}
