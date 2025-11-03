/* ВАРИАНТ №2 - решение задачи l2.14 (объединение done-каналов с итеративным попарным объединением) */

package main

import (
	"fmt"
	"sync"
	"time"
)

// or итеративно сливает каналы в один используя попарное объединение
func or(channels ...<-chan interface{}) <-chan interface{} {

	if len(channels) == 0 {
		return nil
	}

	for len(channels) > 1 {

		merged := make([]<-chan interface{}, 0)

		// идём одновременно с начала и с конца входящего слайса каналов
		for i := 0; i < len(channels)/2; i++ {
			j := len(channels) - 1 - i
			// и объединяем каналы попарно
			merged = append(merged, mergeTwoCannel(channels[i], channels[j]))
		}

		// если остался непарный канал в середине - добавляем его
		if len(channels)%2 == 1 {
			merged = append(merged, channels[len(channels)/2])
		}

		channels = merged
	}

	return channels[0]
}

// mergeTwoCannel объединяет два канала
func mergeTwoCannel(a, b <-chan interface{}) <-chan interface{} {

	out := make(chan interface{})
	// канал сигнала завершения работы всем воркерам
	doneWorkers := make(chan struct{})

	// предусматриваем единственное закрытие канала doneWorkers (чтобы не закрывать закрытый канал)
	var once sync.Once
	closeDoneWorkers := func() { close(doneWorkers) }

	// в отдельной горутине слушаем два канала
	// если что-то приходит по любому из двух каналов, закрываем канал out
	go func() {

		defer close(out) // гарантируем закрытие out

		// слушаем первый канал и канал отмены воркеров
		go func() {
			defer once.Do(closeDoneWorkers)
			select {
			case <-a: // если канал закрылся и получилось его прочитать, case выполняется и горутина завершится
			case <-doneWorkers: // если закрылся канал отмены для горутин и получилось его прочитать, case выполняется и горутина завершится
			}
		}()
		// слушаем второй канал и канал отмены воркеров
		go func() {
			defer once.Do(closeDoneWorkers)
			select {
			case <-b: // если канал закрылся и получилось его прочитать, case выполняется и горутина завершится
			case <-doneWorkers: // если закрылся канал отмены для горутин и получилось его прочитать, case выполняется и горутина завершится
			}
		}()

		// ждём сигнала отмены по всем горутинам
		<-doneWorkers
	}()

	return out
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
	// из канала, программа продолжит выполнение
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
