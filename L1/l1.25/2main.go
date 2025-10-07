/* ВАРИАНТ №2 - решение задачи l1.25 с каналом и горутиной */

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

	// создаём канал отмены, закрытие которого будет снимать блокировку
	done := make(chan struct{})

	// используем time.After в отдельной горутине, чтобы подождать время разблокировки канала
	// (хотя time.After и так работает асинхронно, поэтому можно обойтись и без горутины, и без канала)
	go func() {
		<-time.After(time.Duration(duration) * time.Second)
		close(done)
	}()

	<-done // когда канал закроется, его можно будет вычитать, а функция продолжит выполнение

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
