/* ВАРИАНТ №1 - решение задачи l2.15 Minishell */

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

// BuiltinCommand - встроенные команды
type BuiltinCommand func(args []string) error

// builtins - перечень команд
var builtins = map[string]BuiltinCommand{
	"cd":   cdCommand,   // смена текущей директории
	"pwd":  pwdCommand,  // вывод текущей директории
	"echo": echoCommand, // вывод аргументов
	"kill": killCommand, // сигнал завершения процессу с заданным PID
	"ps":   psCommand,   // список запущенных процессов
}

// MiniShell - основная структура шелла
type MiniShell struct {
	running    bool
	background bool
}

func main() {

	miniShell := &MiniShell{running: true}

	// обработка Ctrl+C
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT)
	go func() {
		for range signalCh {
			fmt.Println("[Ctrl+C] Прерывание команды")
		}
	}()

	miniShell.run()
}

// run
func (s *MiniShell) run() {

	reader := bufio.NewReader(os.Stdin)

	for s.running {
		fmt.Print("mysh> ")

		input, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("\nДо свидания!")
				os.Exit(0)
			}
			fmt.Fprintf(os.Stderr, "Ошибка чтения: %v\n", err)
			continue
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		s.executeLine(input)
	}
}

// executeLine
func (s *MiniShell) executeLine(line string) {

	// обработка нескольких команд через ;
	commands := splitCommands(line, ';')

	for _, cmd := range commands {
		cmd = strings.TrimSpace(cmd)
		if cmd == "" {
			continue
		}

		// обработка конвейеров
		if strings.Contains(cmd, "|") {
			s.executePipeline(cmd)
			continue
		}

		// обработка условных операторов
		if strings.Contains(cmd, "&&") || strings.Contains(cmd, "||") {
			s.executeConditional(cmd)
			continue
		}

		// простая команда
		s.executeCommand(cmd)
	}
}

// встроенные команды

// cdCommand - смена директории
func cdCommand(args []string) error {

	if len(args) < 2 {
		return os.Chdir(os.Getenv("HOME"))
	}

	return os.Chdir(args[1])
}

// pwdCommand - вывод местоположения
func pwdCommand(args []string) error {

	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	fmt.Println(dir)

	return nil
}

// echoCommand - выводит аргументы (например, значение переменной окружения)
func echoCommand(args []string) error {

	if len(args) > 1 {
		// подстановка переменных окружения
		for i := 1; i < len(args); i++ {
			arg := args[i]
			if strings.HasPrefix(arg, "$") {
				envName := arg[1:]
				if val, exists := os.LookupEnv(envName); exists {
					fmt.Print(val + " ")
				} else {
					fmt.Print(" ")
				}
			} else {
				fmt.Print(arg + " ")
			}
		}
	}

	fmt.Println()

	return nil
}

// killCommand - вежливо (15) убивает процесс с заданным pid
func killCommand(args []string) error {

	if len(args) < 2 {
		return fmt.Errorf("использование: kill <pid>")
	}

	pid, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("неверный PID: %v", err)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	return process.Signal(syscall.SIGTERM)
}

// psCommand
func psCommand(args []string) error {

	cmd := exec.Command("ps", "aux")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// executeCommand
func (s *MiniShell) executeCommand(cmdLine string) {

	args := parseArgs(cmdLine)
	if len(args) == 0 {
		return
	}

	// проверяем, является ли команда встроенной
	if builtin, ok := builtins[args[0]]; ok {
		if err := builtin(args); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		}
		return
	}

	// внешняя команда
	s.executeExternalCommand(args)
}

// executeExternalCommand
func (s *MiniShell) executeExternalCommand(args []string) {

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Обработка редиректов
	cmd = handleRedirects(cmd, args)

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка запуска: %v\n", err)
		return
	}

	if !s.background {
		if err := cmd.Wait(); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка выполнения: %v\n", err)
		}
	}
}

// executePipeline
func (s *MiniShell) executePipeline(pipeline string) {
	commands := strings.Split(pipeline, "|")
	var prevOut io.Reader = nil
	var cmds []*exec.Cmd

	// Создаем все команды
	for i, cmdStr := range commands {
		args := parseArgs(strings.TrimSpace(cmdStr))
		if len(args) == 0 {
			continue
		}

		cmd := exec.Command(args[0], args[1:]...)
		cmds = append(cmds, cmd)

		if i > 0 {
			if prevOut != nil {
				cmd.Stdin = prevOut
			}
		}

		if i < len(commands)-1 {
			pipeOut, pipeIn, _ := os.Pipe()
			cmd.Stdout = pipeIn
			prevOut = pipeOut
		} else {
			cmd.Stdout = os.Stdout
		}

		cmd.Stderr = os.Stderr
	}

	// Запускаем все команды
	for _, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка запуска: %v\n", err)
			return
		}
	}

	// Ждем завершения
	for _, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка выполнения: %v\n", err)
		}
	}
}

// executeConditional
func (s *MiniShell) executeConditional(cmdLine string) {

	// простая реализация для демонстрации
	// в реальной реализации нужен полноценный парсинг с учетом приоритетов
	if strings.Contains(cmdLine, "&&") {
		parts := strings.Split(cmdLine, "&&")
		for _, part := range parts {
			s.executeCommand(strings.TrimSpace(part))
		}
	} else if strings.Contains(cmdLine, "||") {
		parts := strings.Split(cmdLine, "||")
		for _, part := range parts {
			s.executeCommand(strings.TrimSpace(part))
		}
	}
}

// вспомогательные функции

// parseArgs разбирает на части введённую команду
func parseArgs(line string) []string {

	var args []string
	var current strings.Builder
	inQuotes := false
	quoteChar := byte(0)

	for i := 0; i < len(line); i++ {
		ch := line[i]

		if ch == '"' || ch == '\'' {
			if !inQuotes {
				inQuotes = true
				quoteChar = ch
			} else if ch == quoteChar {
				inQuotes = false
			} else {
				current.WriteByte(ch)
			}
		} else if ch == ' ' && !inQuotes {
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		} else if ch == '\\' && i+1 < len(line) {
			// экранирование
			i++
			current.WriteByte(line[i])
		} else {
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

// splitCommands
func splitCommands(line string, sep rune) []string {

	var result []string
	var current strings.Builder
	inQuotes := false
	quoteChar := byte(0)

	for i := 0; i < len(line); i++ {
		ch := line[i]

		if ch == '"' || ch == '\'' {
			if !inQuotes {
				inQuotes = true
				quoteChar = ch
			} else if ch == quoteChar {
				inQuotes = false
			}
			current.WriteByte(ch)
		} else if rune(ch) == sep && !inQuotes {
			result = append(result, current.String())
			current.Reset()
		} else {
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// handleRedirects
func handleRedirects(cmd *exec.Cmd, args []string) *exec.Cmd {

	for i, arg := range args {
		if arg == ">" && i+1 < len(args) {
			file, err := os.Create(args[i+1])
			if err == nil {
				cmd.Stdout = file
			}
			cmd.Args = args[:i]
		} else if arg == "<" && i+1 < len(args) {
			file, err := os.Open(args[i+1])
			if err == nil {
				cmd.Stdin = file
			}
			cmd.Args = args[:i]
		}
	}
	return cmd
}
