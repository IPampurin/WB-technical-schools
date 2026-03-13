package db

import (
	"context"
	"time"

	"github.com/IPampurin/EventBooker/pkg/domain"
)

// StorageMethods - интерфейс локальной БД
type StorageMethods interface {
	EventsTableMethods
	BookingTableMethods
	UsersTableMethods
}

// EventsTableMethods - методы для работы с таблицей events
type EventsTableMethods interface {

	// EventCreater - создание мероприятия
	EventCreater(ctx context.Context, name string, date time.Time, bookingTTLMinutes, totalSeats, bookingPrice int) (int, error)

	// GetEvents - получение всех предстоящих мероприятий с информацией о свободных местах
	GetEvents(ctx context.Context) ([]*domain.Event, error)

	// GetEventByID - получение события по id
	GetEventByID(ctx context.Context, id int) (*domain.Event, error)
}

// BookingTableMethods - управление таблицей бронирования
type BookingTableMethods interface {

	// SeatReserver - бронирование места на мероприятии
	SeatReserver(ctx context.Context, eventID, userID int, createdAt, expiresAt time.Time) (int, error)

	// GetEventReserveOfUser - получение данных о брони пользователя на мероприятии (да, один юзер - одно место)
	GetEventReserveOfUser(ctx context.Context, eventID, userID int) (int, string, error)

	// ReserveConfirmer - метод оплаты/подтверждения бронирования
	ReserveConfirmer(ctx context.Context, bookingID int) error

	// CancelBooking - отмена брони
	CancelBooking(ctx context.Context, bookingID int) error
}

// UsersTableMethods - управление таблицей с пользователями
type UsersTableMethods interface {

	// RegisterUser - метод для регистрации пользователя
	RegisterUser(ctx context.Context, name, email string) (int, error)

	// GetUserByEmail - возвращает пользователя по email
	GetUserByEmail(ctx context.Context, email string) (int, error)
}
