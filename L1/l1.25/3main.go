/* ВАРИАНТ №3 - решение задачи l1.25 с time.Timer или time.After */

package main

import (
	"fmt"
	"time"
)

// sleep блокирует выполнение main() на duration секунд
func sleep(duration int) {

	fmt.Println("Время начала сна.")
	fmt.Println()
	fmt.Println("Крепкий здоровый сон...")

	// заводим таймер
	timer := time.NewTimer(time.Duration(duration) * time.Second)
	defer timer.Stop() // с версии Go 1.23 можно и не останавливать таймер - GC сам его остановит
	// когда таймер отмерит заданное время и даст вычитать канал, продолжаем работу функции
	<-timer.C

	// или без создания таймера в одну строку
	// <-time.After(time.Duration(duration) * time.Second)

	fmt.Println()
	fmt.Println("Время завершения сна.")
}

func main() {

	// устанавливаем время блокировки горутины main() в секундах
	timeOfSleep := 5

	fmt.Println("Начинаю работать!")
	fmt.Println("Засыпаю...")
	fmt.Println()

	sleep(timeOfSleep) // запускаем блокировку

	fmt.Println()
	fmt.Println("Проснулся...")
	fmt.Println("Снова работаю!")
	fmt.Println("Конец работы!")
}
