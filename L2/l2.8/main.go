package main

import (
	"fmt"
	"os"
	"time"

	"github.com/beevik/ntp"
)

func main() {

	// выполняем запрос к серверу согласно рекомендации пакета ntp
	ntpTime, err := ntp.Time("0.beevik-ntp.pool.ntp.org")
	// если есть ошибка при получении, выводим ошибку в STDERR и завершаемся с ненулевым кодом
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка получения времени: %v\n", err)
		os.Exit(1)
	}

	// days - мапа для перевода дней недели
	days := map[time.Weekday]string{
		time.Monday:    "понедельник",
		time.Tuesday:   "вторник",
		time.Wednesday: "среда",
		time.Thursday:  "четверг",
		time.Friday:    "пятница",
		time.Saturday:  "суббота",
		time.Sunday:    "воскресенье",
	}

	// получаем русское название дня недели
	rusWeekday := days[ntpTime.Weekday()]

	// форматы даты и времени
	datePartFormat := ntpTime.Format("02.01.2006")
	timePartFormat := ntpTime.Format("15:04:05")

	// выводим время в привычном формате
	fmt.Printf("Текущее время (по NTP серверу): %s %s %s\n", datePartFormat, rusWeekday, timePartFormat)
}
