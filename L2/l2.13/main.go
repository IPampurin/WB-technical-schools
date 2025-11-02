/* ВАРИАНТ №1 - решение задачи l2.13 Утилита cut */

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
)

// Config - конфигурация фильтрации
type Config struct {
	fields          string // строковое представление полей (колонок), которые нужно вывести (флаг -f)
	delimiter       string // использовать другой разделитель (символ) (флаг -d)
	separated       bool   // только строки, содержащие разделитель (флаг -s)
	selectedColumns []int  // разобранные номера столбцов для вывода
}

// parseFlags парсит флаги командной строки
func parseFlags() Config {

	config := Config{}

	// устанавливаем сообщение об использовании на случай ошибки в написании флагов при запуске программы
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Используйте: cut [-f fields] [-d delimiter] [-s] [file]\n")
		fmt.Fprintf(os.Stderr, "Флаги:\n")
		flag.PrintDefaults()
	}

	flag.StringVar(&config.fields, "f", "", "select only these fields (e.g., 1,3-5)")           // по умолчанию все столбцы
	flag.StringVar(&config.delimiter, "d", "\t", "use specified delimiter instead of TAB")      // разделитель по умолчанию (табуляция)
	flag.BoolVar(&config.separated, "s", false, "do not print lines not containing delimiters") // только строки, содержащие разделитель

	flag.Parse() // парсим флаги из командной строки (Must be called after all flags are defined and before flags are accessed by the program)

	// если вообще ничего не указано при запуске
	if config.fields == "" && len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	return config
}

// parseFields парсит строку с номерами полей и возвращает отсортированный список уникальных номеров
func parseFields(fieldsStr string) ([]int, error) {

	if fieldsStr == "" {
		// если поля не указаны, просто возвращаем пустой слайс
		return []int{}, nil
	}

	uniqueFields := make(map[int]struct{}) // номера столбцов
	parts := strings.Split(fieldsStr, ",") // для начала смотрим перечисление через запятую

	for _, part := range parts {

		part = strings.TrimSpace(part) // удаляем лишние пробелы

		// если поля указаны интервально
		if strings.Contains(part, "-") {

			// обрабатываем перечисление через "-"
			rangeParts := strings.Split(part, "-")

			// если возле "-" не два числа, считаем это неправильным
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("неверный формат диапазона: %s", part)
			}

			// получаем первое число и проверяем на адекватность
			start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			if err != nil || start < 1 {
				return nil, fmt.Errorf("неверный номер поля: %s", rangeParts[0])
			}

			// получаем второе число и проверяем на адекватность
			end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err != nil || end < 1 {
				return nil, fmt.Errorf("неверный номер поля: %s", rangeParts[1])
			}

			// проверяем на адекватность вводящего команду
			if start > end {
				return nil, fmt.Errorf("неверный диапазон: начало %d больше конца %d", start, end)
			}

			// переносим весь диапазон в мапу с номерами столбцов
			for i := start; i <= end; i++ {
				uniqueFields[i] = struct{}{}
			}

		} else {
			// иначе, если поле указано через запятую, обрабатываем отдельное поле
			field, err := strconv.Atoi(part)
			if err != nil || field < 1 {
				return nil, fmt.Errorf("неверный номер поля: %s", part)
			}
			uniqueFields[field] = struct{}{}
		}
	}

	// собираем ключи из мапы номеров столбцов и сортируем
	result := slices.Sorted(maps.Keys(uniqueFields))

	return result, nil
}

// processLine обрабатывает одну строку
func processLine(line string, config Config) (string, bool) {

	// если установлен флаг -s и строка не содержит разделитель, пропускаем её
	if config.separated && !strings.Contains(line, config.delimiter) {
		return "", false
	}

	// разбиваем строку на поля
	fields := strings.Split(line, config.delimiter)

	// если нет выбранных полей, возвращаем исходную строку
	if len(config.selectedColumns) == 0 {
		return line, true
	}

	// собираем только выбранные поля
	var outputFields []string
	hasValidFields := false
	for _, col := range config.selectedColumns {
		// нумерация полей начинается с 1, а индексы с 0
		if col-1 < len(fields) {
			outputFields = append(outputFields, fields[col-1])
			hasValidFields = true
		}
		// если поле выходит за границы, просто игнорируем (не добавляем)
	}

	// если ни одно из указанных полей не существует в строке, возвращаем пустую строку
	if !hasValidFields {
		return "", true
	}

	// собираем строку обратно с тем же разделителем
	return strings.Join(outputFields, config.delimiter), true
}

// processInput обрабатывает входные данные
func processInput(config Config, input io.Reader) error {

	// создаем сканер с увеличенным буфером для обработки длинных строк
	scanner := bufio.NewScanner(input)

	// увеличиваем максимальный размер буфера для обработки очень длинных строк
	const maxCapacity = 1024 * 1024 * 1024 // 1 ГБ
	buf := make([]byte, 0, 64*1024)        // начальный буфер 64 КБ
	scanner.Buffer(buf, maxCapacity)

	// обрабатываем каждую строку
	for scanner.Scan() {

		line := scanner.Text()

		// разбиваем строку по разделителю и выбираем нужные поля
		// output - обработанная строка для вывода
		// nextOutput - нужно ли выводить эту строку (false если пропускаем из-за флага -s)
		output, nextOutput := processLine(line, config)

		// выводим строку, если она не пустая и nextOutput == true
		if nextOutput && output != "" {
			fmt.Println(output)
		}
	}

	// проверяем ошибки сканирования
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("ошибка чтения: %v", err)
	}

	return nil
}

func main() {

	config := parseFlags()

	// парсим поля для вывода
	selectedColumns, err := parseFields(config.fields)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ошибка парсинга полей: %v\n", err)
		os.Exit(1)
	}
	config.selectedColumns = selectedColumns

	// определяем источник ввода
	var input io.Reader

	// смотрим аргументы, указанные при запуске
	args := flag.Args()
	if len(args) > 0 {
		// если аргумент (файл) при запуске указан, то читаем из файла
		filename := args[0]

		file, err := os.Open(filename)
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

	// обрабатываем ввод
	if err := processInput(config, input); err != nil {
		fmt.Fprintf(os.Stderr, "ошибка обработки: %v\n", err)
		os.Exit(1)
	}
}
