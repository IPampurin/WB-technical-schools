package main

import (
	"testing"
)

// TestParseFields тестирует функцию parseFields с различными входными данными
func TestParseFields(t *testing.T) {

	tests := []struct {
		name     string // название теста
		input    string // входная строка для парсинга
		expected []int  // ожидаемый результат
		wantErr  bool   // должна ли функция вернуть ошибку
	}{
		{
			name:     "один столбец",
			input:    "1",
			expected: []int{1},
			wantErr:  false,
		},
		{
			name:     "несколько столбцов через запятую",
			input:    "1,3,5",
			expected: []int{1, 3, 5},
			wantErr:  false,
		},
		{
			name:     "диапазон столбцов",
			input:    "2-4",
			expected: []int{2, 3, 4},
			wantErr:  false,
		},
		{
			name:     "смешанный формат",
			input:    "1,3-5,7",
			expected: []int{1, 3, 4, 5, 7},
			wantErr:  false,
		},
		{
			name:     "пустая строка",
			input:    "",
			expected: []int{},
			wantErr:  false,
		},
		{
			name:     "дублирующиеся столбцы",
			input:    "1,2,1,3",
			expected: []int{1, 2, 3},
			wantErr:  false,
		},
		{
			name:     "неверный формат диапазона",
			input:    "1-3-5",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "неверный номер столбца",
			input:    "abc",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "отрицательный номер",
			input:    "-1",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "диапазон с началом больше конца",
			input:    "5-3",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "пробелы вокруг чисел",
			input:    " 1 , 2 - 4 , 5 ",
			expected: []int{1, 2, 3, 4, 5},
			wantErr:  false,
		},
	}

	// запускаем каждый тест
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// вызываем тестируемую функцию
			result, err := parseFields(tt.input)

			// проверяем соответствие ошибки ожиданиям
			if (err != nil) != tt.wantErr {
				t.Errorf("parseFields() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// проверяем соответствие результата ожиданиям
			if !slicesEqual(result, tt.expected) {
				t.Errorf("parseFields() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// TestProcessLine тестирует функцию processLine с различными конфигурациями
func TestProcessLine(t *testing.T) {

	tests := []struct {
		name           string // название теста
		line           string // входная строка
		config         Config // конфигурация для обработки
		expectedOutput string // ожидаемый вывод
		expectedBool   bool   // ожидаемый флаг shouldOutput
	}{
		{
			name: "выбор одного столбца",
			line: "name\tage\tcity",
			config: Config{
				delimiter:       "\t",
				selectedColumns: []int{2},
			},
			expectedOutput: "age",
			expectedBool:   true,
		},
		{
			name: "выбор нескольких столбцов",
			line: "Alice\t25\tLondon\tUK",
			config: Config{
				delimiter:       "\t",
				selectedColumns: []int{1, 3},
			},
			expectedOutput: "Alice\tLondon",
			expectedBool:   true,
		},
		{
			name: "столбцы за пределами данных",
			line: "a\tb\tc",
			config: Config{
				delimiter:       "\t",
				selectedColumns: []int{5, 6},
			},
			expectedOutput: "",
			expectedBool:   true,
		},
		{
			name: "флаг -s с разделителем",
			line: "a,b,c",
			config: Config{
				delimiter:       ",",
				separated:       true,
				selectedColumns: []int{2},
			},
			expectedOutput: "b",
			expectedBool:   true,
		},
		{
			name: "флаг -s без разделителя",
			line: "no_delimiter_here",
			config: Config{
				delimiter:       "\t",
				separated:       true,
				selectedColumns: []int{1},
			},
			expectedOutput: "",
			expectedBool:   false,
		},
		{
			name: "без выбранных столбцов",
			line: "full\tline\there",
			config: Config{
				delimiter:       "\t",
				selectedColumns: []int{},
			},
			expectedOutput: "full\tline\there",
			expectedBool:   true,
		},
		{
			name: "часть столбцов за пределами",
			line: "a\tb\tc",
			config: Config{
				delimiter:       "\t",
				selectedColumns: []int{1, 5},
			},
			expectedOutput: "a",
			expectedBool:   true,
		},
		{
			name: "разделитель запятая",
			line: "John,30,Engineer",
			config: Config{
				delimiter:       ",",
				selectedColumns: []int{1, 3},
			},
			expectedOutput: "John,Engineer",
			expectedBool:   true,
		},
		{
			name: "все столбцы за пределами с флагом -s",
			line: "a\tb",
			config: Config{
				delimiter:       "\t",
				separated:       true,
				selectedColumns: []int{5, 6},
			},
			expectedOutput: "",
			expectedBool:   true,
		},
	}

	// запускаем каждый тест
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// вызываем тестируемую функцию
			output, shouldOutput := processLine(tt.line, tt.config)

			// проверяем соответствие вывода ожиданиям
			if output != tt.expectedOutput {
				t.Errorf("processLine() output = %v, expected %v", output, tt.expectedOutput)
			}

			// проверяем соответствие флага shouldOutput ожиданиям
			if shouldOutput != tt.expectedBool {
				t.Errorf("processLine() shouldOutput = %v, expected %v", shouldOutput, tt.expectedBool)
			}
		})
	}
}

// slicesEqual вспомогательная функция для сравнения двух слайсов int
func slicesEqual(a, b []int) bool {

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
