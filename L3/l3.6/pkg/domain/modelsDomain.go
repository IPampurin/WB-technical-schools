package domain

import "time"

const (
	ProductCategory       = "Продукты"
	TransportCategory     = "Транспорт"
	EntertainmentCategory = "Развлечения"
	HealthCategory        = "Здоровье"
	OtherCategory         = "Другое"
)

// Item представляет запись о продаже/покупке
type Item struct {
	ID       int       // уникальный идентификатор записи
	Category string    // категория ("Продукты", "Транспорт", "Развлечения", "Здоровье", "Другое")
	Amount   float64   // сумма
	Date     time.Time // дата в формате YYYY-MM-DD
}

// CategoryAgregation — структура для результата группировки по категориям
type CategoryAgregation struct {
	Category string  // категория
	Sum      float64 // сумма по категории
	Count    int     // количество по категории
}
