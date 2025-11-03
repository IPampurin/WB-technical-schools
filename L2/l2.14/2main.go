/* ВАРИАНТ №2 - решение задачи l2.14 (объединение done-каналов с итеративным попарным объединением) */

package main

import (
	"fmt"
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

	// в отдельной горутине слушаем два канала
	// если что-то приходит по любому из двух каналов, закрываем канал out
	go func() {
		select {
		case _, ok := <-a:
			// проверяем, что канал рабочий, а не просто закрылся
			// (имело бы смысл, если бы по каналу отмены передавалось какое-либо значение)
			if ok {
				break
			}
		case _, ok := <-b:
			// проверяем, что канал рабочий, а не просто закрылся
			// (имело бы смысл, если бы по каналу отмены передавалось какое-либо значение)
			if ok {
				break
			}
		}
		close(out)
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
