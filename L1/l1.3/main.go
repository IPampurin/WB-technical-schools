/*
Реализовать постоянную запись данных в канал (в главной горутине).
Реализовать набор из N воркеров, которые читают данные из этого канала и выводят их в stdout.
Программа должна принимать параметром количество воркеров и при старте создавать указанное число горутин-воркеров.
*/

package main

import (
	"context"
	"fmt"
	"time"
)

const n = 5 // необходимое количество воркеров

func worker(ch chan any) {

	for v := range ch {
		fmt.Println(v)
	}
}

func main() {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	in := make(chan any, n)

	go func(ctx context.Context) {
		for i := 0; i < 10; i++ {
			in <- i
		}
	}(ctx)

	for i := 0; i < n; i++ {
		go worker(in)
	}

	time.Sleep(5 * time.Second)
}
