/* ВАРИАНТ №1 - решение задачи l1.25 с циклом и проверкой времени */

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

	// засекаем текущее время
	fallAsleepTime := time.Now()
	// определяем в какой момент времени блокировку надо снять
	wakeUpTime := fallAsleepTime.Add(time.Duration(duration) * time.Second)

	// крутим цикл пока текущее время ранее времни снятия блокировки
	var currentTime time.Time
	for currentTime.Before(wakeUpTime) {
		currentTime = time.Now()
	}

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
