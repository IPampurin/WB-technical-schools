package service

import (
	"context"
	"time"

	"github.com/IPampurin/EventBooker/pkg/domain"
	"github.com/wb-go/wbf/logger"
)

// AdminMethods - админское управление событиями
type AdminMethods interface {

	// EventCreater - создание мероприятия
	EventCreater(ctx context.Context, name string, date time.Time, bookingTTLMinutes, totalSeats, bookingPrice int, log logger.Logger) (int, error)
}

// EventMethods - пользовательская работа с событиями
type EventMethods interface {

	// GetEvents - получение всех предстоящих мероприятий с информацией о свободных местах
	GetEvents(ctx context.Context, log logger.Logger) ([]*domain.Event, error)

	// GetEventByID - получение события по id
	GetEventByID(ctx context.Context, id int, log logger.Logger) (*domain.Event, error)
}

// BookerMethods - управление бронированием
type BookerMethods interface {

	// SeatReserver - бронирование места на мероприятии
	SeatReserver(ctx context.Context, eventID, userID int, createdAt time.Time, log logger.Logger) (int, error)

	// GetEventReserveOfUser - получение данных о брони пользователя на мероприятии (да, один юзер - одно место)
	GetEventReserveOfUser(ctx context.Context, eventID, userID int, log logger.Logger) (int, string, error)

	// ReserveConfirmer - метод оплаты/подтверждения бронирования
	ReserveConfirmer(ctx context.Context, bookingID int, log logger.Logger) error

	// CancelBooking - отмена брони
	CancelBooking(ctx context.Context, bookingID int, log logger.Logger) error
}

// ManageUsers - управление пользователями
type ManageUsers interface {

	// RegisterUser - метод для регистрации пользователя
	RegisterUser(ctx context.Context, name, email string, log logger.Logger) (int, error)

	// LoginUser проверяет наличие пользователя с указанным email и возвращает его ID
	LoginUser(ctx context.Context, email string, log logger.Logger) (int, error)
}
