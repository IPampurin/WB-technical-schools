package or_test

import (
	"fmt"
	"time"

	"github.com/IPampurin/MergeDoneChannels/or"
)

// пример использования функции Or
func ExampleOr() {

	// вспомогательная функция для создания канала, который закроется через заданное время
	sig := func(after time.Duration) <-chan interface{} {
		c := make(chan interface{})
		go func() {
			defer close(c)
			time.Sleep(after)
		}()
		return c
	}

	start := time.Now()
	<-or.Or(
		sig(2*time.Hour),
		sig(5*time.Minute),
		sig(1*time.Second),
		sig(1*time.Hour),
		sig(1*time.Minute),
	)

	// округляем время до секунд для стабильного вывода
	fmt.Printf("done after %v", time.Since(start).Round(time.Second))
	// Output: done after 1s
}
