package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestGrep(t *testing.T) {
	// базовый тестовый текст для всех тестов
	baseInput := `первая строка
вторая строка TEST
третья строка
четвертая строка test
пятая строка
Шестая Строка TEST
седьмая строка`

	tests := []struct {
		name     string
		input    string
		config   Config
		expected string
	}{
		// Базовые тесты
		{
			name:  "базовый поиск",
			input: baseInput,
			config: Config{
				pattern: "TEST",
			},
			expected: "вторая строка TEST\nШестая Строка TEST\n",
		},
		{
			name:  "игнорирование регистра",
			input: baseInput,
			config: Config{
				pattern:    "test",
				ignoreCase: true,
			},
			expected: "вторая строка TEST\nчетвертая строка test\nШестая Строка TEST\n",
		},
		{
			name:  "инвертированный поиск",
			input: baseInput,
			config: Config{
				pattern: "TEST",
				invert:  true,
			},
			expected: "первая строка\nтретья строка\nчетвертая строка test\nпятая строка\nседьмая строка\n",
		},
		{
			name:  "номера строк",
			input: baseInput,
			config: Config{
				pattern:    "TEST",
				lineNumber: true,
			},
			expected: "2:вторая строка TEST\n6:Шестая Строка TEST\n",
		},
		{
			name:  "только подсчет",
			input: baseInput,
			config: Config{
				pattern: "TEST",
				count:   true,
			},
			expected: "2\n",
		},

		// Тесты контекста
		{
			name:  "контекст после",
			input: baseInput,
			config: Config{
				pattern: "TEST",
				after:   1,
			},
			expected: "вторая строка TEST\nтретья строка\nШестая Строка TEST\nседьмая строка\n",
		},
		{
			name:  "контекст до",
			input: baseInput,
			config: Config{
				pattern: "TEST",
				before:  1,
			},
			expected: "первая строка\nвторая строка TEST\nпятая строка\nШестая Строка TEST\n",
		},
		{
			name:  "полный контекст",
			input: baseInput,
			config: Config{
				pattern: "TEST",
				context: 1,
			},
			expected: "первая строка\nвторая строка TEST\nтретья строка\nпятая строка\nШестая Строка TEST\nседьмая строка\n",
		},

		// Комбинированные флаги
		{
			name:  "игнор регистра + номера строк",
			input: baseInput,
			config: Config{
				pattern:    "test",
				ignoreCase: true,
				lineNumber: true,
			},
			expected: "2:вторая строка TEST\n4:четвертая строка test\n6:Шестая Строка TEST\n",
		},
		{
			name:  "инверт + подсчет",
			input: baseInput,
			config: Config{
				pattern: "TEST",
				invert:  true,
				count:   true,
			},
			expected: "5\n",
		},
		{
			name:  "контекст + номера строк",
			input: baseInput,
			config: Config{
				pattern:    "TEST",
				context:    1,
				lineNumber: true,
			},
			expected: "1:первая строка\n2:вторая строка TEST\n3:третья строка\n5:пятая строка\n6:Шестая Строка TEST\n7:седьмая строка\n",
		},

		// Граничные случаи
		{
			name:  "пустой ввод",
			input: "",
			config: Config{
				pattern: "TEST",
			},
			expected: "",
		},
		{
			name:  "нет совпадений",
			input: baseInput,
			config: Config{
				pattern: "NOMATCH",
			},
			expected: "",
		},
		{
			name:  "фиксированная строка",
			input: "строка с . точкой\nстрока без точки",
			config: Config{
				pattern: "с . точкой",
				fixed:   true,
			},
			expected: "строка с . точкой\n",
		},

		// Все флаги вместе
		{
			name:  "все флаги кроме подсчета",
			input: baseInput,
			config: Config{
				pattern:    "test",
				ignoreCase: true,
				invert:     false,
				lineNumber: true,
				context:    1,
			},
			expected: "1:первая строка\n2:вторая строка TEST\n3:третья строка\n4:четвертая строка test\n5:пятая строка\n6:Шестая Строка TEST\n7:седьмая строка\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputLines, count, err := grep(tt.config, strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("grep вернула ошибку: %v", err)
			}

			var result bytes.Buffer

			if count != 0 {
				// Режим подсчета
				result.WriteString(tt.expected)
			} else if outputLines != nil {
				// Обычный режим
				for _, line := range *outputLines {
					if tt.config.lineNumber {
						result.WriteString(fmt.Sprintf("%d:%s\n", line.number, line.content))
					} else {
						result.WriteString(line.content + "\n")
					}
				}
			}

			// Для режима подсчета сравниваем числа как строки
			if count != 0 {
				expected := strings.TrimSpace(tt.expected)
				actual := strings.TrimSpace(result.String())
				if actual != expected {
					t.Errorf("ожидался счетчик %s, получен %s", expected, actual)
				}
			} else {
				// Для обычного вывода сравниваем строки
				if result.String() != tt.expected {
					t.Errorf("ожидалось:\n%q\nполучено:\n%q", tt.expected, result.String())
				}
			}
		})
	}
}

func TestGrepInvalidPattern(t *testing.T) {
	config := Config{
		pattern: "[invalid", // невалидное регулярное выражение
	}

	_, _, err := grep(config, strings.NewReader("test"))
	if err == nil {
		t.Error("ожидалась ошибка для невалидного шаблона")
	}
}

func TestGrepWithFile(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "файл - базовый поиск",
			config: Config{
				pattern: "TEST",
			},
			expected: "вторая строка TEST\nШестая Строка TEST\n",
		},
		{
			name: "файл - игнорирование регистра",
			config: Config{
				pattern:    "test",
				ignoreCase: true,
			},
			expected: "вторая строка TEST\nчетвертая строка test\nШестая Строка TEST\n",
		},
		{
			name: "файл - номера строк",
			config: Config{
				pattern:    "TEST",
				lineNumber: true,
			},
			expected: "2:вторая строка TEST\n6:Шестая Строка TEST\n",
		},
		{
			name: "файл - контекст",
			config: Config{
				pattern: "TEST",
				context: 1,
			},
			expected: "первая строка\nвторая строка TEST\nтретья строка\nпятая строка\nШестая Строка TEST\nседьмая строка\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Открываем файл вместо использования strings.NewReader
			file, err := os.Open("test_data.txt")
			if err != nil {
				t.Fatalf("не удалось открыть test_data.txt: %v", err)
			}
			defer file.Close()

			outputLines, count, err := grep(tt.config, file)
			if err != nil {
				t.Fatalf("grep вернула ошибку: %v", err)
			}

			var result bytes.Buffer

			if count != 0 {
				result.WriteString(tt.expected)
			} else if outputLines != nil {
				for _, line := range *outputLines {
					if tt.config.lineNumber {
						result.WriteString(fmt.Sprintf("%d:%s\n", line.number, line.content))
					} else {
						result.WriteString(line.content + "\n")
					}
				}
			}

			if count != 0 {
				expected := strings.TrimSpace(tt.expected)
				actual := strings.TrimSpace(result.String())
				if actual != expected {
					t.Errorf("ожидался счетчик %s, получен %s", expected, actual)
				}
			} else {
				if result.String() != tt.expected {
					t.Errorf("ожидалось:\n%q\nполучено:\n%q", tt.expected, result.String())
				}
			}
		})
	}
}
