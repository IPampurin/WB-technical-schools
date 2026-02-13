package storage

import "time"

// Event описывает запись в календаре событий
type Event struct {
	ID      int       `json:"id"`                // id события (счётчик событий)
	UserID  int       `json:"user_id"`           // id пользователя
	Date    time.Time `json:"date"`              // дата события
	Title   string    `json:"title"`             // заголовок события
	Content string    `json:"content,omitempty"` // содержание события
}

// Repository - интерфейс, реализующий требуемые методы
type Repository interface {
	Create(userID int, date time.Time, title, content string) (int, error) // добавляет event в хранилище, возвращает ID event или ошибку
	Update(event *Event) error                                             // обновляет event в хранилище, возвращает ошибку, если событие не найдено
	Delete(userID, eventID int) error                                      // удаляет event из хранилища, возвращает ошибку, если событие не найдено
	GetForDay(userID int, date time.Time) ([]*Event, error)                // возвращает перечень событий на день или ошибку
	GetForWeek(userID int, date time.Time) ([]*Event, error)               // возвращает перечень событий на неделю или ошибку
	GetForMonth(userID int, date time.Time) ([]*Event, error)              // возвращает перечень событий на месяц или ошибку
}
