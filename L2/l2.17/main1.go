package main

import (
	"bufio"
	"context"
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

// sigScan слушает сигналы отмены
func sigScan(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup) {

	defer wg.Done()

	// заводим и регистрируем канал для обработки Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)
	defer signal.Stop(sigChan)

	select {
	case <-ctx.Done():
		fmt.Println("sigScan: Отмена контекста.")
		return
	case sig := <-sigChan:
		fmt.Printf("sigScan: Сигнал завершения программы: %v.\n", sig)
		cancel()
		return
	}
}

func main() {

	// считываем данные запуска
	address, timeout := startRead()

	// устанавливаем соединение
	conn, err := net.DialTimeout(networkDefault, address, timeout)
	if err != nil {
		fmt.Printf("Не удалось подключиться по указанному адресу %s, ошибка:%v.\n", address, err)
		return
	}
	defer conn.Close()

	fmt.Printf("Подключено к %s\n", address)
	fmt.Println("Ctrl+D для выхода, Ctrl+C для прерывания")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// запускаем обработку сигналов отмены
	wg.Add(1)
	go sigScan(ctx, cancel, &wg)

	// запускаем горутину считывания консоли
	wg.Add(1)
	go inputReadAndSend(ctx, cancel, conn, &wg)

	// запускаем горутину чтения данных с сервера
	wg.Add(1)
	go serverReadAndWrite(ctx, cancel, conn, &wg)

	// дожидаемся завершения
	wg.Wait()

	fmt.Println("Программа завершена.")
}

// inputReadAndSend читает из консоли и отправляет серверу
func inputReadAndSend(ctx context.Context, cancel context.CancelFunc, conn net.Conn, wg *sync.WaitGroup) {

	defer func() {
		wg.Done()
		conn.Close()
	}()

	r := bufio.NewReader(os.Stdin)
	w := bufio.NewWriter(conn)

	rw := bufio.NewReadWriter(r, w)

	buf := make([]byte, 1024) // буфер

	for {
		select {
		// либо прерываемся по отмене контекста
		case <-ctx.Done():
			fmt.Println("inputReadAndSend: Отмена контекста, завершение чтения ввода.")
			return
		// либо считываем консоль и отправляем данные серверу
		default:
			n, err := rw.Read(buf)
			if err != nil {
				if err == io.EOF {
					// если словили io.EOF, значит получили Ctrl+D
					fmt.Println("\ninputReadAndSend: Выход из программы.")
					rw.Flush() // сбрасываем буфер для порядка
					cancel()
					return
				}
				fmt.Fprintf(os.Stderr, "inputReadAndSend: Ошибка чтения из консоли: %v\n", err)
				rw.Flush() // сбрасываем буфер для порядка
				cancel()
				return
			}

			// если ошибок нет, пишем в conn
			_, err = rw.Write(buf[:n])
			if err != nil {
				fmt.Fprintf(os.Stderr, "inputReadAndSend: Ошибка записи в соединение: %v\n", err)
				return
			}

			rw.Flush() // сбрасываем буфер
		}
	}
}

// serverReadAndWrite принимает данные с сервера и выводит в консоль
func serverReadAndWrite(ctx context.Context, cancel context.CancelFunc, conn net.Conn, wg *sync.WaitGroup) {

	defer func() {
		wg.Done()
		conn.Close()
	}()

	r := bufio.NewReader(conn)
	w := bufio.NewWriter(os.Stdout)

	rw := bufio.NewReadWriter(r, w)

	buf := make([]byte, 1024) // буфер

	for {
		select {
		// либо прерываемся по отмене контекста
		case <-ctx.Done():
			fmt.Println("serverReadAndWrite: Отмена контекста, завершение чтения соединения.")
			return
		// либо считываем данные с сервера и выводим в Stdout
		default:
			n, err := rw.Read(buf)
			if err != nil {
				if err == io.EOF {
					fmt.Println("serverReadAndWrite: Соединение закрыто сервером.")
				} else {
					fmt.Fprintf(os.Stderr, "serverReadAndWrite: Ошибка чтения соединения: %v\n", err)
				}
				rw.Flush() // сбрасываем буфер для порядка
				cancel()
				return
			}

			// если ошибок нет, пишем в Stdout
			_, err = rw.Write(buf[:n])
			if err != nil {
				fmt.Fprintf(os.Stderr, "serverReadAndWrite: Ошибка вывода в Stdout: %v\n", err)
				rw.Flush() // сбрасываем буфер для порядка
				cancel()
				return
			}

			rw.Flush() // сбрасываем буфер
		}
	}
}
