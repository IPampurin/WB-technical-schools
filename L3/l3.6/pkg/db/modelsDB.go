package db

import "time"

// Item представляет запись в таблице items
type Item struct {
	ID       int       `db:"id"`       // уникальный идентификатор записи
	Category string    `db:"category"` // категория ("Продукты", "Транспорт", "Развлечения", "Здоровье", "Другое")
	Amount   float64   `db:"amount"`   // сумма
	Date     time.Time `db:"date"`     // дата в формате YYYY-MM-DD
}

// CategoryAgregation — структура для результата группировки по категориям
type CategoryAgregation struct {
	Category string  `db:"category"` // категория
	Sum      float64 `db:"sum"`      // сумма по категории
	Count    int     `db:"count"`    // количество по категории
}
