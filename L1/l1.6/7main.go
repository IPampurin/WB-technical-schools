/* ВАРИАНТ №7 - корректная остановка при поступлении сигнала ОС */

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// doSomething удваивает поступающие числа
func doSomething(ctx context.Context, in <-chan int, out chan<- int, wg *sync.WaitGroup) {

	defer wg.Done()  // вычитаем счетчик WaitGroup
	defer close(out) // закрываем out в любом случае

	for {
		select {
		case v, ok := <-in: // слушаем in, и удваиваем пришедшие числа
			if !ok {
				fmt.Println("Канал входных данных закрыт. Горутина завершила работу.")
				return
			}
			time.Sleep(500 * time.Millisecond) // создаём вид бурной дейтельности
			// но из-за этой паузы возникает ситуация, когда main() уже не слушает out и можем
			// получить dedlock при отправке результата. Поэтому перед отправкой выполяем проверку
			select {
			case out <- v * 2: // если это возможно, отправляем результат
			case <-ctx.Done(): // а если время вышло, то закрываем канал и выходим
				fmt.Println("Получен сигнал отмены по контексту в горутине. Горутина завершила работу.")
				return
			}
		case <-ctx.Done(): // при поступлении сигнала завершаем горутину
			fmt.Println("Получен сигнал отмены по контексту в горутине. Горутина завершила работу.")
			return
		}
	}
}

func main() {

	// определяем контекст
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // гарантируем отмену контекста

	in := make(chan int)  // канал отправки данных в работу
	out := make(chan int) // канал приёма результатов

	// канал получения сигналов ОС, буфер 1 для гарантии вычитывания
	sigChan := make(chan os.Signal, 1)
	// регистрируем канал для получения сигналов прерывания программы пользователем (Ctrl+C)
	// или мягкого завершения окружением
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	wg.Add(1)                         // добавляем счетчик WaitGroup
	go doSomething(ctx, in, out, &wg) // запускаем сравнение в отдельной горутине

	number := 0 // числа для отправки в работу

loop:
	for {
		select {
		case in <- number: // если можем, отправляем число в работу
			number++
		case v, ok := <-out: // если в out появляется результат, выводим его после проверки условия
			if !ok {
				break loop // если out закрыт, выходим из цикла
			}
			fmt.Println(v)
		case <-sigChan: // при получении сигнала отмены выходим из цикла
			fmt.Println("Получен сигнал ОС на завершение в main().")
			break loop
		}
	}

	cancel() // сигнал горутине закругляться
	// close(in) // закрытие in избыточно - закроется сам
	wg.Wait() // ждём остановки горутины

	fmt.Println("Завершили main().")
}
