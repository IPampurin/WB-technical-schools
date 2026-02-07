/* ВАРИАНТ №2 - решение задачи l2.17 */

package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	hostDefault    = "tcpbin.com"     // хост по умолчанию для теста
	portDefault    = "4242"           // порт по умолчанию для теста
	timeoutDefault = 10 * time.Second // таймаут по умолчанию
	networkDefault = "tcp"            // протокол по умолчанию
)

// startRead возвращает хост:порт и таймаут назначенные при запуске, либо по умолчанию
func startRead() (string, time.Duration) {

	// функция комментариев
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Использование: %s [--timeout=10s] [хост порт]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Примеры:\n")
		fmt.Fprintf(os.Stderr, "  %s --timeout=10s tcpbin.com 4242\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s tcpbin.com 4242 (использует значение по умолчанию: --timeout=%v)\n",
			os.Args[0], timeoutDefault)
		fmt.Fprintf(os.Stderr, "  %s (использует значения по умолчанию: --timeout=%v %s:%s)\n",
			os.Args[0], timeoutDefault, hostDefault, portDefault)
	}

	// устанавливаем значения по умолчанию
	host := hostDefault
	port := portDefault

	// парсим ввод команды
	timeout := flag.Duration("timeout", timeoutDefault, "Таймаут соединения")

	// парсим флаг
	flag.Parse()

	// получаем позиционные аргументы
	args := flag.Args()

	// обрабатываем переданные аргументы
	switch len(args) {
	case 0:
		// ничего не передано - используем значения по умолчанию
		fmt.Printf("Используются значения по умолчанию: --timeout=%v %s:%s\n", timeout, host, port)
	case 2:
		// только хост и порт
		host = args[0]
		port = args[1]
	default:
		// другое количество аргументов
		fmt.Fprintf(os.Stderr, "Ошибка: не корректные аргументы.\n")
		flag.Usage()
		os.Exit(1)
	}

	// формируем адрес
	address := net.JoinHostPort(host, port)

	return address, *timeout
}

func main() {

	// считываем данные запуска
	address, timeout := startRead()

	// устанавливаем соединение
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		fmt.Printf("Ошибка подключения: %v\n", err)
		return
	}
	defer conn.Close()

	done := make(chan struct{}) // канал для закрытия горутины обработки сигналов

	// обеспечиваем закрытие соединения, хотя
	// повторное закрытие соединения и безопасно
	var closeOnce sync.Once
	closeConn := func() {
		// игнорируем низкоуровневые системные ошибки
		// (их возникновение почти невозможно в типовых сценариях)
		_ = conn.Close()
		fmt.Println("Соединение закрыто")
		close(done)
	}

	fmt.Printf("Подключено к %s\n", address)
	fmt.Println("Ctrl+D для выхода, Ctrl+C для прерывания")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)

	var wg sync.WaitGroup

	// обработка сигналов
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-sigChan:
			fmt.Println("Получен сигнал завершения программы.")
			closeOnce.Do(closeConn)
		case <-done:
			fmt.Println("Получен сигнал done.")
		}
	}()

	// STDIN -> Сокет
	wg.Add(1)
	go func() {
		defer wg.Done()
		// перекладываем данные в conn из os.Stdin
		// при любых ошибках завершаем работу
		_, _ = io.Copy(conn, os.Stdin)
		closeOnce.Do(closeConn)
	}()

	// Сокет -> STDOUT
	wg.Add(1)
	go func() {
		defer wg.Done()
		// перекладываем данные в os.Stdin из conn
		// при любых ошибках завершаем работу
		_, _ = io.Copy(os.Stdout, conn)
		closeOnce.Do(closeConn)
	}()

	wg.Wait()

	fmt.Println("\nПрограмма завершена")
}

/*
// что делает oi.Copy
func simpleCopy(dst io.Writer, src io.Reader) error {
    buf := make([]byte, 32*1024) // буфер
    for {
        n, err := src.Read(buf)
        if n > 0 {
            _, writeErr := dst.Write(buf[:n])
            if writeErr != nil {
                return writeErr
            }
        }
        if err != nil {
            if err == io.EOF {
                return nil
            }
            return err
        }
    }
}
*/
