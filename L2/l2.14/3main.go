/* ВАРИАНТ №3 - решение задачи l2.14 (объединение done-каналов с рекурсивным попарным объединением) */

package main

import (
	"fmt"
	"sync"
	"time"
)

// or рекурсивно сливает каналы в один используя попарное объединение
func or(channels ...<-chan interface{}) <-chan interface{} {

	// если каналов нет, возвращаем nil
	if len(channels) == 0 {
		return nil
	}
	// если остался один канал, его и возвращаем
	if len(channels) == 1 {
		return channels[0]
	}

	// делим слайс пополам и используем рекурсию для каждой половины
	mid := len(channels) / 2
	left := or(channels[:mid]...)
	right := or(channels[mid:]...)

	return mergeTwoChannel(left, right)
}

// mergeTwoChannel объединяет два канала
func mergeTwoChannel(a, b <-chan interface{}) <-chan interface{} {

	out := make(chan interface{})
	// канал сигнала завершения работы всем воркерам
	doneWorkers := make(chan struct{})

	// предусматриваем единственное закрытие канала doneWorkers (чтобы не закрывать закрытый канал)
	var once sync.Once
	closeDoneWorkers := func() { close(doneWorkers) }

	// в отдельной горутине слушаем два канала
	// если что-то приходит по любому из двух каналов, закрываем канал out и
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
