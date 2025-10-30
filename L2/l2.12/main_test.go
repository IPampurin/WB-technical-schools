package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// TestGrepBasic тестирует базовые случаи поиска
func TestGrepBasic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		config   Config
		expected string
	}{
		{
			name:  "базовый поиск",
			input: "hello world\ntest pattern\nanother line\npattern matching\nlast line",
			config: Config{
				pattern: "pattern",
			},
			expected: "test pattern\npattern matching\n",
		},
		{
			name:  "поиск с игнорированием регистра",
			input: "hello world\ntest pattern\nanother line",
			config: Config{
				pattern:    "HELLO",
				ignoreCase: true,
			},
			expected: "hello world\n",
		},
		{
			name:  "инвертированный поиск",
			input: "hello world\ntest pattern\nanother line\nlast line",
			config: Config{
				pattern: "pattern",
				invert:  true,
			},
			expected: "hello world\nanother line\nlast line\n",
		},
		{
			name:  "поиск с номерами строк",
			input: "hello world\ntest pattern\nanother line\npattern matching\nlast line",
			config: Config{
				pattern:    "pattern",
				lineNumber: true,
			},
			expected: "2:test pattern\n4:pattern matching\n",
		},
		{
			name:  "только подсчёт",
			input: "hello world\ntest pattern\nanother line\npattern matching\nlast line",
			config: Config{
				pattern: "pattern",
				count:   true,
			},
			expected: "2\n",
		},
		{
			name:  "фиксированная строка",
			input: "hello world\ntest pattern\nanother line",
			config: Config{
				pattern: "patt.rn",
				fixed:   true,
			},
			expected: "", // не должно быть совпадений, т.к. точка экранирована
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// временно перехватываем stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// запускаем grep
			err := grep(tt.config, strings.NewReader(tt.input))

			// восстанавливаем stdout
			w.Close()
			os.Stdout = oldStdout

			if err != nil {
				t.Fatalf("grep вернула ошибку: %v", err)
			}

			// читаем вывод
			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if output != tt.expected {
				t.Errorf("ожидалось:\n%q\nполучено:\n%q", tt.expected, output)
			}
		})
	}
}

// TestGrepContext тестирует контекст (флаги -A, -B, -C)
func TestGrepContext(t *testing.T) {
	input := `line 1
line 2
MATCH 1
line 4
line 5
MATCH 2
line 7`

	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "контекст после",
			config: Config{
				pattern: "MATCH",
				after:   1,
			},
			expected: "MATCH 1\nline 4\nMATCH 2\nline 7\n",
		},
		{
			name: "контекст до",
			config: Config{
				pattern: "MATCH",
				before:  1,
			},
			expected: "line 2\nMATCH 1\nline 5\nMATCH 2\n",
		},
		{
			name: "полный контекст",
			config: Config{
				pattern: "MATCH",
				context: 1,
			},
			expected: "line 2\nMATCH 1\nline 4\nline 5\nMATCH 2\nline 7\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := grep(tt.config, strings.NewReader(input))

			w.Close()
			os.Stdout = oldStdout

			if err != nil {
				t.Fatalf("grep вернула ошибку: %v", err)
			}

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if output != tt.expected {
				t.Errorf("ожидалось:\n%q\nполучено:\n%q", tt.expected, output)
			}
		})
	}
}

// TestGrepEdgeCases тестирует граничные случаи
func TestGrepEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		config   Config
		expected string
	}{
		{
			name:  "пустой ввод",
			input: "",
			config: Config{
				pattern: "pattern",
			},
			expected: "",
		},
		{
			name:  "нет совпадений",
			input: "строка 1\nстрока 2\nстрока 3",
			config: Config{
				pattern: "pattern",
			},
			expected: "",
		},
		{
			name:  "все строки совпадают",
			input: "pattern 1\npattern 2\npattern 3",
			config: Config{
				pattern: "pattern",
			},
			expected: "pattern 1\npattern 2\npattern 3\n",
		},
		{
			name:  "контекст больше чем строки",
			input: "line 1\nMATCH\nline 3",
			config: Config{
				pattern: "MATCH",
				before:  10,
				after:   10,
			},
			expected: "line 1\nMATCH\nline 3\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := grep(tt.config, strings.NewReader(tt.input))

			w.Close()
			os.Stdout = oldStdout

			if err != nil {
				t.Fatalf("grep вернула ошибку: %v", err)
			}

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if output != tt.expected {
				t.Errorf("ожидалось:\n%q\nполучено:\n%q", tt.expected, output)
			}
		})
	}
}

// TestGrepWithTestFile тестирует с реальным файлом
func TestGrepWithTestFile(t *testing.T) {
	// создаем временный файл для тестирования
	testContent := `Первая строка файла
Вторая строка с текстом
Третья строка содержит PATTERN для поиска
Четвертая строка после паттерна
Пятая строка тоже имеет pattern в тексте
Шестая строка без совпадений
Седьмая строка PATTERN в верхнем регистре
Восьмая строка завершающая`

	tmpfile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("не удалось создать временный файл: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.WriteString(testContent); err != nil {
		t.Fatalf("не удалось записать в временный файл: %v", err)
	}
	tmpfile.Close()

	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "поиск PATTERN в файле",
			config: Config{
				pattern: "PATTERN",
			},
			expected: "Третья строка содержит PATTERN для поиска\nСедьмая строка PATTERN в верхнем регистре\n",
		},
		{
			name: "поиск pattern с игнорированием регистра",
			config: Config{
				pattern:    "pattern",
				ignoreCase: true,
			},
			expected: "Третья строка содержит PATTERN для поиска\nПятая строка тоже имеет pattern в тексте\nСедьмая строка PATTERN в верхнем регистре\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := os.Open(tmpfile.Name())
			if err != nil {
				t.Fatalf("не удалось открыть тестовый файл: %v", err)
			}
			defer file.Close()

			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err = grep(tt.config, file)

			w.Close()
			os.Stdout = oldStdout

			if err != nil {
				t.Fatalf("grep вернула ошибку: %v", err)
			}

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if output != tt.expected {
				t.Errorf("ожидалось:\n%q\nполучено:\n%q", tt.expected, output)
			}
		})
	}
}

// TestGrepInvalidPattern тестирует обработку неверных шаблонов
func TestGrepInvalidPattern(t *testing.T) {
	config := Config{
		pattern: "[invalid regex", // незакрытая скобка
	}

	err := grep(config, strings.NewReader("test line"))

	if err == nil {
		t.Error("ожидалась ошибка для неверного шаблона, но ошибки нет")
	}

	if !strings.Contains(err.Error(), "неверный шаблон") {
		t.Errorf("ожидалась ошибка про неверный шаблон, получили: %v", err)
	}
}
