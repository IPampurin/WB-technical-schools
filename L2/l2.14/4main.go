/* ВАРИАНТ №4 - решение задачи l2.14 (объединение done-каналов с пулом воркеров 2) */

package main

import (
	"fmt"
	"sync"
	"time"
)

// or возвращает канал, который можно прочитать после закрытия любого из каналов отмены
func or(channels ...<-chan interface{}) <-chan interface{} {

	outCh := make(chan interface{})    // возвращаемый канал
	doneWorkers := make(chan struct{}) // канал завершения работы всем воркерам

	// предусматриваем единственное закрытие канала doneWorkers (чтобы не закрывать закрытый канал)
	var once sync.Once
	closeDoneWorkers := func() { close(doneWorkers) }

	go func() {

		defer close(outCh) // закрываем отдаваемый канал, чтобы в main из него можно было словить zero value

		for _, ch := range channels {
			go func(c <-chan interface{}) {
				<-c // просто ждем закрытия канала
				once.Do(closeDoneWorkers)
			}(ch)
		}

		<-doneWorkers // ждем сигнала отмены
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
