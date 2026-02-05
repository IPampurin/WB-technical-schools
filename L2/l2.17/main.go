package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

/*
Примеры серверов для теста:
smtp.mail.ru:25 (может быть заблокирован)
smtp.mail.ru:587
smtp.yandex.ru:587
echo.websocket.org:80 - WebSocket echo
*/
const (
	hostDefault    = "echo.websocket.org"
	portDefault    = "80"
	timeOutDefault = 10 * time.Second
	networkDefault = "tcp"
)

// Config хранит параметры, задаваемые пользователем при запуске клиента
type Config struct {
	network string
	host    string
	port    string
	timeOut time.Time
}

// Client - telnet клиент
type Client struct {
	ctx    context.Context
	cancel context.CancelFunc
	conn   *net.Conn
	Config
}

// newClient возвращает экземпляр клиента
func newClient() *Client {

	return &Client{}
}

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host := hostDefault
	port := portDefault
	timeOut := timeOutDefault
	network := networkDefault

	address := host + ":" + port
	conn, err := net.DialTimeout(network, address, timeOut)
	if err != nil {
		fmt.Printf("Не удалось подключиться по указанному адресу %s, ошибка:%v.\n", address, err)
		return
	}
	defer conn.Close()

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

	var wg sync.WaitGroup

	wg.Add(1)
	go readInputAndSend(ctx, conn, &wg) // считываем консоль и отправляем данные

	wg.Add(1)
	go writeMessage(ctx, conn, &wg) // выводим в консоль пришедшее

	wg.Wait()
}

func readInputAndSend(ctx context.Context, conn net.Conn, wg *sync.WaitGroup) {

	defer func() {
		wg.Done()
	}()

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
		}
	}
}

func sendToServerWithRetry(conn net.Conn, data []byte) error {

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

func writeMessage(ctx context.Context, conn net.Conn, wg *sync.WaitGroup) {

	defer wg.Done()

	scanner := bufio.NewScanner(conn)
	w := bufio.NewWriter(os.Stdout)

	for scanner.Scan() {
		fmt.Fprint(w, "\n"+string(scanner.Bytes())+"\n")
		w.Flush()
	}
}
