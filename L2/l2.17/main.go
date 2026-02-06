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
	// ничего не передано - используем значения по умолчанию
	case 0:
		fmt.Printf("Используются значения по умолчанию: --timeout=%v %s:%s\n", timeout, host, port)
	// только хост и порт
	case 2:
		host = args[0]
		port = args[1]
	// слишком много аргументов
	default:
		fmt.Fprintf(os.Stderr, "Ошибка: слишком много аргументов.\n")
		flag.Usage()
		os.Exit(1)
	}

	// формируем адрес
	address := net.JoinHostPort(host, port)

	return address, *timeout
}

// Client - telnet клиент
type Client struct {
	ctx    context.Context
	cancel context.CancelFunc
	conn   net.Conn
}

func sigScan(ctx context.Context, cancel context.CancelFunc) {

	// заводим и регистрируем канал для обработки Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)

	// горутина для обработки Ctrl+C
	go func() {
		for {
			select {
			case <-ctx.Done():
				signal.Stop(sigChan)
				return
			case sig := <-sigChan:
				fmt.Printf("Поступил сигнал завершения работы программы: %v.\n", sig)
				cancel()
				return
			}
		}
	}()
}

func main() {

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

	// запускаем обработку сигналов отмены
	go sigScan(ctx, cancel)

	// определяем экземпляр клиента
	client := Client{
		ctx:    ctx,
		cancel: cancel,
		conn:   conn,
	}

	// запускаем горутину считывания консоли
	go readInputAndSend(ctx, conn)

	// вычитываем данные из соединения
	scanner := bufio.NewScanner(conn)
	w := bufio.NewWriter(os.Stdout)

	for scanner.Scan() {
		fmt.Fprint(w, "\n"+string(scanner.Bytes())+"\n")
		w.Flush()
	}

	fmt.Println("Программа завершена.")
}

func readInputAndSend(ctx context.Context, conn net.Conn) {

	scanner := bufio.NewScanner(os.Stdin)

	for {
		// выводим приглашение командной строки
		fmt.Print("Введите данные> ")

		select {
		// либо прерываемся по отмене контекста
		case <-ctx.Done():
			return
		// либо считываем консоль и отправляем данные серверу
		default:
			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					fmt.Fprintf(os.Stderr, "Ошибка чтения из консоли: %v\n", err)
					continue
				}
				// Ctrl+D даёт io.EOF, что воспринимаем как сигнал к остановке программы
				fmt.Println("\nexit")
				return
			}

			if _, err := conn.Write(scanner.Bytes()); err != nil {
				if errors.Is(err, net.ErrClosed) || errors.Is(err, os.ErrClosed) {
					fmt.Fprintf(os.Stderr, "Соединение прервано, ошибка: %v\n", err)
					return
				}
			}

			fmt.Println("Сообщение ", scanner.Text(), " отправлено.")
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
