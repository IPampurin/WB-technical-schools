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

// builtins - мапа команд
var builtins = map[string]BuiltinCommand{
	"cd":   cdCom,
	"pwd":  pwdCom,
	"echo": echoCom,
	"kill": killCom,
	"ps":   psCom,
	"exit": exitCom,
}

// Shell - структура шелла
type Shell struct {
	lastStatus     int
	currentProcess *os.Process
}

func main() {

	// заводим экземпляр
	shell := &Shell{
		lastStatus:     0,
		currentProcess: nil,
	}

	// запускаем
	runShell(shell)
}

// runShell - основной цикл утилиты
func runShell(shell *Shell) {

	// заводим и регистрируем канал для обработки Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)

	// горутина для обработки Ctrl+C
	go func() {
		for sig := range sigChan {
			if shell.currentProcess != nil {
				// если есть запущенный процесс - убиваем его
				shell.currentProcess.Signal(sig)
			} else {
				// если нет процесса - просто печатаем ^C и новую строку
				fmt.Println("\n^C")
				fmt.Print("mysh> ")
			}
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		// выводим приглашение командной строки
		fmt.Print("mysh> ")

		// считываем строку
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				fmt.Fprintf(os.Stderr, "Ошибка чтения: %v\n", err)
				continue
			}
			// EOF - Ctrl+D был нажат
			fmt.Println("\nexit")
			os.Exit(0)
		}

		line := scanner.Text()

		// убираем пробелы
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		runCommand(shell, line)
	}
}

// runCommand - основная логика выполнения
func runCommand(shell *Shell, line string) {

	// разделяем на команды по ; (с учётом того, что ; может стоять внутри команды "echo "hello; world"")
	commands := splitSeparator(line, ';')
	for _, cmd := range commands {
		cmd = strings.TrimSpace(cmd)
		if cmd == "" {
			continue
		}

		// обработка условных операторов && и ||
		if strings.Contains(cmd, "&&") || strings.Contains(cmd, "||") {
			executeConditional(shell, cmd)
			continue
		}

		// обработка конвейеров
		if strings.Contains(cmd, "|") {
			status := executePipeline(shell, cmd)
			shell.lastStatus = status
			continue
		}

		// выполнение простой команды
		status := executeSimpleCommand(shell, cmd)
		shell.lastStatus = status
	}
}

// splitSeparator делит строку по разделителю с учётом кавычек
func splitSeparator(line string, sep rune) []string {

	var result []string         // список команд
	var current strings.Builder // буфер для накопления текущей команды
	inQuotes := false           // флаг нахождения внутри кавычек
	quoteChar := rune(0)        // символ кавычки, которая открыта (' или ")

	for _, ch := range line {
		if ch == '"' || ch == '\'' {
			// если мы не внутри кавычек, значит это начало кавычек
			if !inQuotes {
				inQuotes = true
				quoteChar = ch
				// если мы внутри кавычек и символ совпадает с открывающей кавычкой
			} else if ch == quoteChar {
				// значит это конец кавычек
				inQuotes = false
			}
			// записываем кавычку в буфер
			current.WriteRune(ch)
			// если символ - разделитель и мы не внутри кавычек
		} else if !inQuotes && ch == sep {
			// добавляем текущую команду в результат, убирая пробелы по краям
			result = append(result, strings.TrimSpace(current.String()))
			current.Reset() // очищаем буфер для следующей команды
		} else {
			// не кавычки просто добавляем в буфер
			current.WriteRune(ch)
		}
	}

	// если после обработки всей строки в буфере что-то осталось
	if current.Len() > 0 {
		// добавляем последнюю команду в результат
		result = append(result, strings.TrimSpace(current.String()))
	}

	return result
}

// executeConditional проверяет выполнение команд с учётом условных операторов && и ||
func executeConditional(shell *Shell, line string) bool {

	// разбиваем строку на части по операторам && и || с учётом кавычек
	parts := splitConditional(line)
	if len(parts) == 0 {
		return false
	}

	// флаг, указывающий, нужно ли продолжать выполнение следующих команд в цепочке
	executeNext := true
	// сохраняем статус последней выполненной команды из состояния shell
	lastStatus := shell.lastStatus

	for i := 0; i < len(parts); i++ {
		part := parts[i]
		// если текущая часть - оператор (&& или ||), пропускаем его
		if part == "&&" || part == "||" {
			continue
		}

		// определяем оператор, который стоит после текущей команды
		operator := ""
		if i+1 < len(parts) {
			operator = parts[i+1]
		}

		// определяем, нужно ли выполнять текущую команду
		// изначально считаем, что выполнять нужно, если не было прерывания цепочки
		shouldExecute := executeNext
		if operator == "&&" {
			// для && выполняем, если предыдущая команда завершилась успешно (статус 0)
			shouldExecute = shouldExecute && (lastStatus == 0)
		} else if operator == "||" {
			// для || выполняем, если предыдущая команда завершилась с ошибкой (статус не 0)
			shouldExecute = shouldExecute && (lastStatus != 0)
		}

		// если нужно выполнить команду
		if shouldExecute {
			// выполняем команду и получаем её статус
			status := executeSimpleCommand(shell, part)
			lastStatus = status
			shell.lastStatus = status

			// определяем, нужно ли выполнять следующие команды
			if operator == "&&" && status != 0 {
				// если оператор && и команда завершилась с ошибкой, останавливаем выполнение
				executeNext = false
			} else if operator == "||" && status == 0 {
				// если оператор || и команда завершилась успешно, останавливаем выполнение
				executeNext = false
			}
		} else {
			// если команда не выполняется, устанавливаем статус ошибки
			lastStatus = 1
			shell.lastStatus = 1
		}

		// если следующий элемент - оператор, пропускаем его
		if i+1 < len(parts) && (parts[i+1] == "&&" || parts[i+1] == "||") {
			i++
		}
	}

	// возвращаем флаг, указывающий, были ли выполнены какие-либо команды
	return executeNext
}

// splitConditional разбивает строку на части по операторам && и || с учётом кавычек
func splitConditional(line string) []string {

	var result []string         // список команд
	var current strings.Builder // буфер для накопления текущей команды
	inQuotes := false           // флаг нахождения внутри кавычек
	quoteChar := rune(0)        // символ кавычки, которая открыта (' или ")

	runes := []rune(line)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]

		if ch == '"' || ch == '\'' {
			// если мы не внутри кавычек, значит это начало кавычек
			if !inQuotes {
				inQuotes = true
				quoteChar = ch
				// если мы внутри кавычек и символ совпадает с открывающей кавычкой
			} else if ch == quoteChar {
				// значит это конец кавычек
				inQuotes = false
			}
			// записываем кавычку в буфер
			current.WriteRune(ch)
			continue
		}

		// если мы не внутри кавычек и есть следующий символ
		if !inQuotes && i+1 < len(runes) {
			next := runes[i+1]

			// проверяем, является ли текущий и следующий символ оператором &&
			if ch == '&' && next == '&' {

				// если в буфере есть накопленная команда, добавляем её в результат
				if current.Len() > 0 {
					result = append(result, current.String())
					current.Reset()
				}

				// добавляем оператор && в результат
				result = append(result, "&&")
				i++ // пропускаем следующий символ, так как он уже обработан как часть оператора
				continue
			}

			// проверяем, является ли текущий и следующий символ оператором ||
			if ch == '|' && next == '|' {

				// если в буфере есть накопленная команда, добавляем её в результат
				if current.Len() > 0 {
					result = append(result, current.String())
					current.Reset()
				}

				// добавляем оператор || в результат
				result = append(result, "||")
				// пропускаем следующий символ, так как он уже обработан как часть оператора
				i++
				continue
			}
		}

		// если символ не кавычка и не является частью оператора, добавляем его в буфер
		current.WriteRune(ch)
	}

	// если после обработки всей строки в буфере что-то осталось
	if current.Len() > 0 {
		// добавляем последнюю команду в результат
		result = append(result, current.String())
	}

	return result
}

// executeSimpleCommand выполняет единичную команду
func executeSimpleCommand(shell *Shell, cmdLine string) int {

	// парсим строку команды: разбиваем на аргументы, получаем файлы для редиректов
	args, stdinFile, stdoutFile, stderrFile, stdoutAppend := parseCommand(cmdLine)
	// если команда пустая (нет аргументов), ничего не делаем
	if len(args) == 0 {
		return 0
	}

	// проверяем, является ли команда встроенной (cd, echo ...)
	if builtin, ok := builtins[args[0]]; ok {

		// сохраняем оригинальные стандартные потоки
		oldStdin := os.Stdin
		oldStdout := os.Stdout
		oldStderr := os.Stderr

		// функция для восстановления потоков
		defer func() {
			os.Stdin = oldStdin
			os.Stdout = oldStdout
			os.Stderr = oldStderr
		}()

		// обработка редиректа ввода
		if stdinFile != "" {
			file, err := os.Open(stdinFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ошибка открытия файла %s: %v\n", stdinFile, err)
				return 1
			}
			defer file.Close()
			os.Stdin = file
		}

		// обработка редиректа вывода
		if stdoutFile != "" {
			var file *os.File
			var err error
			if stdoutAppend {
				file, err = os.OpenFile(stdoutFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			} else {
				file, err = os.Create(stdoutFile)
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "ошибка создания файла %s: %v\n", stdoutFile, err)
				return 1
			}
			defer file.Close()
			os.Stdout = file
		}

		// обработка редиректа ошибок
		if stderrFile != "" {
			file, err := os.Create(stderrFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ошибка создания файла %s: %v\n", stderrFile, err)
				return 1
			}
			defer file.Close()
			os.Stderr = file
		}

		// выполняем встроенную команду
		if err := builtin(args); err != nil {
			fmt.Fprintf(os.Stderr, "ошибка: %v\n", err)
			return 1
		}
		return 0
	}

	// если команда не встроенная, запускаем внешнюю команду
	return executeExternalCommand(shell, args, stdinFile, stdoutFile, stderrFile, stdoutAppend)
}

// parseCommand разбивает команду на аргументы, получает файлы для редиректов
func parseCommand(line string) (args []string, stdinFile, stdoutFile, stderrFile string, stdoutAppend bool) {

	// сначала разбиваем на токены с учетом кавычек
	tokens := tokenizeLine(line)

	// обработка токенов: разделяем команды и редиректы
	var cmdTokens []string // список аргументов команды (без редиректов)
	i := 0
	for i < len(tokens) {
		token := tokens[i]

		// проверяем на редирект ввода ( < filename )
		if token == "<" && i+1 < len(tokens) {
			stdinFile = tokens[i+1]
			i += 2 // пропускаем и оператор редиректа, и имя файла
		} else if token == ">" && i+1 < len(tokens) {
			// редирект вывода ( > filename )
			stdoutFile = tokens[i+1]
			i += 2
		} else if token == "2>" && i+1 < len(tokens) {
			// редирект вывода ошибок ( 2> filename )
			stderrFile = tokens[i+1]
			i += 2
		} else if token == ">>" && i+1 < len(tokens) {
			// редирект вывода с добавлением ( >> filename )
			stdoutFile = tokens[i+1]
			stdoutAppend = true // запоминаем, что нужен режим добавления
			i += 2
			/*// для >> открываем файл в режиме добавления
			if file, err := os.OpenFile(stdoutFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
				file.Close() // сразу закрываем, команда откроет файл заново
			}*/
		} else {
			// если это не редирект, добавляем токен к аргументам команды
			cmdTokens = append(cmdTokens, token)
			i++
		}
	}

	// подстановка переменных окружения
	for i, token := range cmdTokens {
		if strings.HasPrefix(token, "$") {
			envName := token[1:] // убираем символ $
			if val, exists := os.LookupEnv(envName); exists {
				cmdTokens[i] = val // заменяем на значение переменной
			} else {
				cmdTokens[i] = "" // если переменная не определена, заменяем на пустую строку
			}
		}
	}

	// возвращаем аргументы команды и имена файлов для редиректов
	return cmdTokens, stdinFile, stdoutFile, stderrFile, stdoutAppend
}

// tokenizeLine разбивает строку на токены с учетом кавычек
func tokenizeLine(line string) []string {

	var tokens []string         // список токенов (аргументов)
	var current strings.Builder // буфер для накопления текущего токена
	inQuotes := false           // флаг нахождения внутри кавычек
	quoteChar := byte(0)        // символ кавычки, которая открыта (' или ")

	for i := 0; i < len(line); i++ {
		ch := line[i]

		if ch == '"' || ch == '\'' {
			// если мы не внутри кавычек, значит это начало кавычек
			if !inQuotes {
				inQuotes = true
				quoteChar = ch
				// если мы внутри кавычек и символ совпадает с открывающей кавычкой
			} else if ch == quoteChar {
				// значит это конец кавычек
				inQuotes = false
			}
			// не записываем кавычку в токен
			continue

		}

		// если символ - пробел или табуляция и мы не внутри кавычек
		if !inQuotes && (ch == ' ' || ch == '\t') {
			// если в буфере есть накопленный токен, добавляем его в список
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset() // очищаем буфер для следующего токена
			}
			// пробелы и табуляции не сохраняются в токенах
		} else {
			// все остальные символы (не кавычки и не пробелы вне кавычек) добавляем в буфер
			current.WriteByte(ch)
		}
	}

	// если после обработки всей строки в буфере что-то осталось
	if current.Len() > 0 {
		// добавляем последний токен в список
		tokens = append(tokens, current.String())
	}

	return tokens
}

// executeExternalCommand выполняет внешнюю команду
func executeExternalCommand(shell *Shell, args []string, stdinFile, stdoutFile, stderrFile string, stdoutAppend bool) int {

	// создаём команду для выполнения: первый аргумент - имя программы, остальные - её аргументы
	cmd := exec.Command(args[0], args[1:]...)

	// переменные для хранения открытых файлов
	var stdinFileHandle, stdoutFileHandle, stderrFileHandle *os.File

	// настройка стандартных потоков
	cmd.Stdin = os.Stdin   // ввод с клавиатуры
	cmd.Stdout = os.Stdout // вывод на экран
	cmd.Stderr = os.Stderr // ошибки на экран

	// обработка редиректов ввода ( < file )
	if stdinFile != "" {
		file, err := os.Open(stdinFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ошибка открытия файла %s: %v\n", stdinFile, err)
			return 1
		}
		stdinFileHandle = file
		cmd.Stdin = file              // перенаправляем ввод из файла
		defer stdinFileHandle.Close() // закрываем файл после выполнения команды
	}

	// обработка редиректа вывода ( > file )
	if stdoutFile != "" {
		var file *os.File
		var err error
		if stdoutAppend {
			file, err = os.OpenFile(stdoutFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		} else {
			file, err = os.Create(stdoutFile)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "ошибка создания файла %s: %v\n", stdoutFile, err)
			// если stdin файл был открыт, он закроется в defer
			return 1
		}
		stdoutFileHandle = file
		cmd.Stdout = file              // перенаправляем вывод в файл
		defer stdoutFileHandle.Close() // закрываем файл после выполнения команды
	}

	// обработка редиректа ошибок ( 2> file )
	if stderrFile != "" {
		file, err := os.Create(stderrFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ошибка создания файла %s: %v\n", stderrFile, err)
			return 1
		}
		stderrFileHandle = file
		cmd.Stderr = file              // перенаправляем ошибки в файл
		defer stderrFileHandle.Close() // закрываем файл после выполнения команды
	}

	// запускаем команду (но не ждём её завершения)
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "ошибка запуска: %v\n", err)
		return 1
	}

	// сохраняем процесс для обработки Ctrl+C (чтобы можно было прервать команду)
	shell.currentProcess = cmd.Process

	// ждем завершения команды
	if err := cmd.Wait(); err != nil {
		// если команда завершилась с ненулевым кодом возврата
		if exitErr, ok := err.(*exec.ExitError); ok {
			shell.currentProcess = nil // очищаем ссылку на процесс
			return exitErr.ExitCode()  // возвращаем код завершения команды
		}
		// если произошла другая ошибка при ожидании
		fmt.Fprintf(os.Stderr, "ошибка выполнения: %v\n", err)
		shell.currentProcess = nil
		return 1
	}

	// команда завершилась успешно (код 0)
	shell.currentProcess = nil
	return 0
}

// executePipeline обрабатывает конвейер команд
func executePipeline(shell *Shell, pipeline string) int {

	// разбиваем строку на отдельные команды по символу '|'
	commands := strings.Split(pipeline, "|")
	// если команд нет, ничего не делаем
	if len(commands) == 0 {
		return 0
	}

	// массив для хранения объектов команд
	cmds := make([]*exec.Cmd, len(commands))
	// переменные для хранения открытых файлов
	var openedFiles []*os.File
	// переменная для хранения вывода предыдущей команды (для связи команд через pipe)
	var prevOut io.ReadCloser

	// обрабатываем команды по очереди
	for i, cmdStr := range commands {

		// убираем пробелы вокруг команды
		cmdStr = strings.TrimSpace(cmdStr)
		// получаем аргументы и файлы для редиректов
		args, stdinFile, stdoutFile, stderrFile, stdoutAppend := parseCommand(cmdStr)
		// если команда пустая (например, "|" или "||"), выдаём ошибку
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "ошибка: пустая команда в конвейере")
			return 1
		}

		// создаём объект команды
		cmd := exec.Command(args[0], args[1:]...)
		// сохраняем команду в массив
		cmds[i] = cmd

		// обработка редиректа ошибок (stderr)
		if stderrFile != "" {
			file, err := os.Create(stderrFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ошибка создания файла: %v\n", err)
				return 1
			}
			openedFiles = append(openedFiles, file) // сохраняем файл для последующего закрытия
			cmd.Stderr = file                       // перенаправляем stderr команды в файл
		} else {
			cmd.Stderr = os.Stderr // иначе отправляем ошибки в стандартный поток ошибок
		}

		// обработка ввода (stdin)
		if i == 0 {
			// для первой команды в пайплайне
			if stdinFile != "" {
				// если указан редирект ввода из файла
				file, err := os.Open(stdinFile)
				if err != nil {
					fmt.Fprintf(os.Stderr, "ошибка открытия файла: %v\n", err)
					return 1
				}
				openedFiles = append(openedFiles, file) // сохраняем файл для последующего закрытия
				cmd.Stdin = file                        // читаем stdin из файла
			} else {
				cmd.Stdin = os.Stdin // иначе читаем из стандартного ввода
			}
		} else {
			cmd.Stdin = prevOut // для последующих команд stdin это вывод предыдущей команды
		}

		// обработка вывода (stdout)
		if i == len(commands)-1 {
			// для последней команды в пайплайне
			if stdoutFile != "" {
				// если указан редирект вывода в файл
				var file *os.File
				var err error
				if stdoutAppend {
					file, err = os.OpenFile(stdoutFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				} else {
					file, err = os.Create(stdoutFile)
				}
				if err != nil {
					fmt.Fprintf(os.Stderr, "ошибка создания файла: %v\n", err)
					return 1
				}
				openedFiles = append(openedFiles, file) // сохраняем файл для последующего закрытия
				cmd.Stdout = file                       // записываем вывод в файл
			} else {
				cmd.Stdout = os.Stdout // иначе выводим в стандартный вывод
			}
		} else {
			// для промежуточных команд создаём pipe для связи со следующей командой
			reader, writer := io.Pipe()
			// stdout текущей команды пишет в writer
			cmd.Stdout = writer
			// следующая команда будет читать из reader
			prevOut = reader
		}
	}

	// функция для закрытия всех открытых файлов
	defer func() {
		for _, file := range openedFiles {
			if file != nil {
				file.Close()
			}
		}
	}()

	// запускаем все команды пайплайна (не ждём завершения)
	for _, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "ошибка запуска команды: %v\n", err)
			return 1
		}
	}

	// сохраняем последний процесс для обработки Ctrl+C
	if len(cmds) > 0 && cmds[len(cmds)-1].Process != nil {
		shell.currentProcess = cmds[len(cmds)-1].Process
	}

	// ждем завершения всех команд
	var lastStatus int
	for i, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				if i == len(cmds)-1 {
					lastStatus = exitErr.ExitCode() // сохраняем статус последней команды в пайплайне
				}
			}
		}
		// закрываем writer после завершения команды (кроме последней)
		if i < len(cmds)-1 {
			if w, ok := cmd.Stdout.(*io.PipeWriter); ok {
				w.Close()
			}
		}
	}

	// очищаем ссылку на процесс
	shell.currentProcess = nil

	// возвращаем статус завершения последней команды в пайплайне
	return lastStatus
}

// встроенные команды

func cdCom(args []string) error {

	var dir string
	if len(args) < 2 {

		var err error
		dir, err = os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cd: не удалось получить домашнюю директорию: %v", err)
		}

	} else {

		dir = args[1]
		if dir == "~" {

			var err error
			dir, err = os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("cd: %v", err)
			}

		} else if strings.HasPrefix(dir, "~/") {

			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("cd: %v", err)
			}

			dir = home + dir[1:]
		}
	}

	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("cd: %s: %v", dir, err)
	}

	return nil
}

func pwdCom(args []string) error {

	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	fmt.Println(dir)

	return nil
}

func echoCom(args []string) error {

	if len(args) > 1 {

		output := make([]string, 0, len(args)-1)
		for i := 1; i < len(args); i++ {
			arg := args[i]
			// подстановка переменных окружения
			if strings.HasPrefix(arg, "$") {
				val := os.Getenv(arg[1:])
				output = append(output, val)
			} else {
				output = append(output, arg)
			}
		}

		fmt.Println(strings.Join(output, " "))

	} else {
		fmt.Println()
	}

	return nil
}

func killCom(args []string) error {

	if len(args) < 2 {
		return fmt.Errorf("использование: kill <pid>")
	}

	pid, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("неверный PID: %v", err)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("процесс не найден: %v", err)
	}

	return process.Signal(syscall.SIGTERM)
}

func psCom(args []string) error {

	cmd := exec.Command("ps", "aux")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func exitCom(args []string) error {

	fmt.Println("exit")
	os.Exit(0)

	return nil
}
