package domain

import "time"

// Event - информация о мероприятии
type Event struct {
	ID                int       // id события
	Name              string    // название события
	DateEvent         time.Time // время проведения события
	BookingTTLMinutes int       // время в минутах, на которое можно сделать бронь
	TotalSeats        int       // количество мест всего
	FreeSeats         int       // количество свободных мест
	BookedSeats       int       // количество забронированных мест (активных броней)
	BookingPrice      int       // стоимость, которую надо внести при бронировании (если 0, то предоплата не требуется)
}

// User - пользователь (не админ)
type User struct {
	ID    int    // id пользователя
	Name  string // имя пользователя
	Email string // почта для внешней идентификации
}

const (
	BookingStatusPending   = "pending"   // ожидает оплаты/подтверждения
	BookingStatusConfirmed = "confirmed" // оплачено/подтверждено
	BookingStatusCancelled = "cancelled" // отменено (вручную или по таймеру)
)

// Booking - информация о брони
type Booking struct {
	ID          int        // id брони
	EventID     int        // id мероприятия
	UserID      int        // id пользователя, сделавшего бронь
	Status      string     // статус: pending, confirmed, cancelled
	CreatedAt   time.Time  // время создания брони
	ExpiresAt   time.Time  // время, до которого нужно подтвердить (для pending)
	ConfirmedAt *time.Time // время подтверждения оплатой (nil, если ещё не подтверждено)
}
