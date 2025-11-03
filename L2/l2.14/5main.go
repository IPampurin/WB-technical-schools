/* ВАРИАНТ №5 - решение задачи l2.14 (объединение done-каналов с reflect.Select) */

package main

import (
	"fmt"
	"reflect"
	"time"
)

// or возвращает канал, который можно прочитать после закрытия любого из каналов отмены
func or(channels ...<-chan interface{}) <-chan interface{} {

	outCh := make(chan interface{}) // возвращаемый канал

	// запускаем горутину, которая будет следить за всеми каналами
	go func() {

		defer close(outCh) // закрываем отдаваемый канал, чтобы в main из него можно было словить zero value

		// создаем слайс SelectCase для функции reflect.Select
		// SelectCase - это структура, описывающая один case в операторе select
		cases := make([]reflect.SelectCase, len(channels))

		// заполняем cases для каждого переданного канала
		for i, ch := range channels {
			cases[i] = reflect.SelectCase{
				// направление - получение (чтение из канала)
				// (SelectRecv - чтение, SelectSend - запись, SelectDefault - default case)
				Dir: reflect.SelectRecv,
				// сам канал, обернутый в reflect.Value, reflect.ValueOf(ch) превращает
				// обычный канал в объект рефлексии, с которым может работать reflect.Select
				Chan: reflect.ValueOf(ch),
			}
		}

		// reflect.Select - это динамический аналог оператора select
		// Он блокируется до тех пор, пока не сработает один из cases
		// В нашем случае мы ждем, когда можно будет прочитать из любого канала
		//
		// Что происходит когда канал закрывается:
		// - Операция <-ch на закрытом канале сразу возвращает zero value
		// - reflect.Select понимает это и разблокируется
		//
		// Функция возвращает:
		// - chosen: индекс case который сработал
		// - recv: значение которое было прочитано (нам не нужно)
		// - recvOK: было ли значение действительно отправлено (false если канал закрыт)
		reflect.Select(cases)
		// как только любой канал закрывается - выходим и закрываем out в defer
	}()

	return outCh
}

func main() {

	// sig - функция производства каналов
	sig := func(after time.Duration) <-chan interface{} {
		c := make(chan interface{})
		go func() {
			defer close(c)
			time.Sleep(after)
		}()
		return c
	}

	// начинаем слушать канал
	start := time.Now()
	// как только появится возможность прочитать
	// из канала or, программа продолжит выполнение
	<-or(
		sig(2*time.Hour),
		sig(5*time.Minute),
		sig(1*time.Second),
		sig(1*time.Hour),
		sig(1*time.Minute),
	)

	// печатаем время прослушки канала
	fmt.Printf("done after %v", time.Since(start))
}
