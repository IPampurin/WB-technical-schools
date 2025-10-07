/* ВАРИАНТ №4 - решение задачи l1.25 с контекстом */

package main

import (
	"context"
	"fmt"
	"time"
)

// sleep блокирует выполнение main() на duration секунд
func sleep(duration int) {

	fmt.Println("Время начала сна.")
	fmt.Println()
	fmt.Println("Крепкий здоровый сон...")

	// организуем контекст с таймаутом через duration секунд
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(duration)*time.Second)
	defer cancel()

	<-ctx.Done() // получаем сигнал отмены контекста и функция может работать дальше

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
