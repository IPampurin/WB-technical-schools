package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
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

func sigScan(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup) {

	defer wg.Done()

	// заводим и регистрируем канал для обработки Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)

	for {
		select {
		case <-ctx.Done():
			signal.Stop(sigChan)
			fmt.Println("sigScan: Отмена контекста.")
			return
		case sig := <-sigChan:
			fmt.Printf("sigScan: Сигнал завершения программы: %v.\n", sig)
			cancel()
			return
		}
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// запускаем обработку сигналов отмены
	wg.Add(1)
	go sigScan(ctx, cancel, &wg)
	/*
		// определяем экземпляр клиента
		client := Client{
			ctx:    ctx,
			cancel: cancel,
			conn:   conn,
		}
	*/
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

	defer wg.Done()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		select {
		// либо прерываемся по отмене контекста
		case <-ctx.Done():
			fmt.Println("inputReadAndSend: Отмена контекста, завершение чтения ввода.")
			return
		// либо считываем консоль и отправляем данные серверу
		default:
			// выводим приглашение командной строки
			fmt.Print("Данные для отправки серверу: ")

			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					fmt.Fprintf(os.Stderr, "inputReadAndSend: Ошибка чтения из консоли: %v\n", err)
					continue
				}
				// Ctrl+D (Ctrl+Z+Enter для Win) даёт io.EOF (при этом scanner.Scan() == false, а err == nil),
				// что воспринимаем как сигнал к остановке программы
				fmt.Println("\ninputReadAndSend: Выход из программы.")
				cancel()
				return
			}

			message := scanner.Text()
			// добавляем \r\n для совместимости с большинством протоколов
			// (HTTP, SMTP, FTP и др. требуют \r\n)
			if !strings.HasSuffix(message, "\n") {
				message += "\r\n"
			}

			if _, err := conn.Write([]byte(message)); err != nil {
				if errors.Is(err, net.ErrClosed) || errors.Is(err, os.ErrClosed) {
					fmt.Fprintf(os.Stderr, "inputReadAndSend: Соединение прервано, ошибка: %v\n", err)
					cancel()
					return
				}
			}
		}
	}
}

// serverReadAndWrite принимает данные с сервера и выводит в консоль
func serverReadAndWrite(ctx context.Context, cancel context.CancelFunc, conn net.Conn, wg *sync.WaitGroup) {

	defer wg.Done()

	// reader := bufio.NewReader(conn)

	scanner := bufio.NewScanner(conn)
	w := bufio.NewWriter(os.Stdout)

	for {
		select {
		// либо прерываемся по отмене контекста
		case <-ctx.Done():
			fmt.Println("serverReadAndWrite: Отмена контекста, завершение чтения данных от сервера.")
			return
		// либо считываем данные с сервера и выводим в Stdout
		default:
			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					fmt.Fprintf(os.Stderr, "serverReadAndWrite: Ошибка чтения данных от сервера: %v\n", err)
					cancel()
					return
				}
				// io.EOF == закрытие соединения (err == nil),
				// что воспринимаем как сигнал к остановке программы
				fmt.Println("\nserverReadAndWrite: Сервер закрыл соединение.")
				fmt.Println("serverReadAndWrite: Выход из программы.")
				cancel()
				return
			}

			fmt.Fprintf(w, "Сервер: %s\n", string(scanner.Bytes()))
			w.Flush()
		}
	}
}

/*
// newClient возвращает экземпляр клиента
func newClient(conn *net.Conn) *Client {

		ctx, cancel := context.WithCancel(context.Background())

		return &Client{
			ctx:    ctx,
			cancel: cancel,
			conn:   *conn,
		}
	}
*/

/*
func sendToServerWithRetry(data []byte) error {

	_, err := conn.Write(data)
	if err == nil {
		return nil
	}

	if errors.Is(err, net.ErrClosed) || errors.Is(err, os.ErrClosed) {

	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка отправки сообщения : %v\n", err)
	}

	return nil
}
*/
