/*
Реализовать постоянную запись данных в канал (в главной горутине).
Реализовать набор из N воркеров, которые читают данные из этого канала и выводят их в stdout.
Программа должна принимать параметром количество воркеров и при старте создавать указанное число горутин-воркеров.
*/

package main

import (
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

	in := make(chan any, n)

	go func() {
		for i := 0; i < 10; i++ {
			in <- i
		}
	}()

	for i := 0; i < n; i++ {
		go worker(in)
	}

	time.Sleep(5 * time.Second)
}
