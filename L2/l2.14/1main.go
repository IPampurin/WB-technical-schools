/* ВАРИАНТ №1 - решение задачи l2.14 (объединение done-каналов с пулом воркеров) */

package main

import (
	"fmt"
	"sync"
	"time"
)

// or сливает все каналы в один
func or(channels ...<-chan interface{}) <-chan interface{} {

	outCh := make(chan interface{})    // возвращаемый канал
	doneWorkers := make(chan struct{}) // канал завершения работы всем воркерам

	// предусматриваем единственное закрытие канала doneWorkers (чтобы не закрывать закрытый канал)
	var once sync.Once
	closeDoneWorkers := func() { close(doneWorkers) }

	var wg sync.WaitGroup

	go func() {

		defer close(outCh) // закрываем отдаваемый канал, чтобы в main из него можно было словить zero value

		// объявим пул воркеров и дадим каждому воркеру по каналу из channels
		wg.Add(len(channels))
		for i := 0; i < len(channels); i++ {

			// в воркере будем слушать каналы
			go func(i int) {
				defer wg.Done()
				select {
				// если получаем сигнал отмены из внешнего канала отмены
				// отправляем сигнал отмены для остальных воркеров
				case <-channels[i]:
					once.Do(closeDoneWorkers) // один раз закрываем канал оповещения остальных горутин
				// а если получаем сигнал о том, что кто-то из воркеров уже получил
				// внешний сигнал отмены, завершаем работу воркера
				case <-doneWorkers:
				}
			}(i)
		}
		wg.Wait()
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
