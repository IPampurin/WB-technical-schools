package db

import "time"

type Event struct {
	ID                int       `db:"id"`                  // id события
	Name              string    `db:"name"`                // название события
	DateEvent         time.Time `db:"date_event"`          // время проведения события
	BookingTTLMinutes int       `db:"booking_ttl_minutes"` // время в минутах, на которое можно сделать бронь
	TotalSeats        int       `db:"total_seats"`         // количество мест всего
	FreeSeats         int       `db:"free_seats"`          // количество свободных мест
	BookingPrice      int       `db:"booking_price"`       // стоимость, которую надо внести при бронировании
}

type User struct {
	ID    int    `db:"id"`    // id пользователя
	Name  string `db:"name"`  // имя пользователя
	Email string `db:"email"` // почта для внешней идентификации
}

type Booking struct {
	ID          int        `db:"id"`           // id брони
	EventID     int        `db:"event_id"`     // id мероприятия
	UserID      int        `db:"user_id"`      // id пользователя, сделавшего бронь
	Status      string     `db:"status"`       // статус: pending, confirmed, cancelled
	CreatedAt   time.Time  `db:"created_at"`   // время создания брони
	ExpiresAt   time.Time  `db:"expires_at"`   // время, до которого нужно подтвердить (для pending)
	ConfirmedAt *time.Time `db:"confirmed_at"` // время подтверждения оплатой (nil, если ещё не подтверждено)
}
