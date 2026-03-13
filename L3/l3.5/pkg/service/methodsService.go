package service

import (
	"context"
	"fmt"
	"time"

	"github.com/IPampurin/EventBooker/pkg/domain"
	"github.com/wb-go/wbf/logger"
)

// EventCreater - создание мероприятия
func (s *Service) EventCreater(ctx context.Context, name string, date time.Time, bookingTTLMinutes, totalSeats, bookingPrice int, log logger.Logger) (int, error) {

	if date.Before(time.Now()) {
		return 0, fmt.Errorf("дата мероприятия не может быть в прошлом")
	}

	id, err := s.storage.EventCreater(ctx, name, date, bookingTTLMinutes, totalSeats, bookingPrice)
	if err != nil {
		return 0, fmt.Errorf("ошибка EventCreater при создании события: %w", err)
	}

	log.Info("мероприятие создано", "id", id, "name", name)

	return id, nil
}

// GetEvents - получение всех предстоящих мероприятий с информацией о свободных местах
func (s *Service) GetEvents(ctx context.Context, log logger.Logger) ([]*domain.Event, error) {

	events, err := s.storage.GetEvents(ctx)
	if err != nil {
		return nil, fmt.Errorf("ошибка GetEvents при получении событий: %w", err)
	}

	log.Info("получены мероприятия", "count", len(events))

	return events, nil
}

// GetEventByID - получение события по id
func (s *Service) GetEventByID(ctx context.Context, id int, log logger.Logger) (*domain.Event, error) {

	event, err := s.storage.GetEventByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("ошибка GetEventByID при получении события: %w", err)
	}

	log.Info("мероприятие получено", "id", id)

	return event, nil
}

// SeatReserver - бронирование места на мероприятии
func (s *Service) SeatReserver(ctx context.Context, eventID, userID int, createdAt time.Time, log logger.Logger) (int, error) {

	// 1. Проверяем, нет ли уже у пользователя брони на это событие
	existingBookingID, _, err := s.storage.GetEventReserveOfUser(ctx, eventID, userID)
	if err != nil {
		return 0, fmt.Errorf("ошибка проверки существующей брони: %w", err)
	}
	if existingBookingID != 0 {
		return 0, fmt.Errorf("пользователь уже забронировал место на этом мероприятии")
	}

	// 2. Получаем мероприятие, чтобы узнать время жизни брони
	event, err := s.storage.GetEventByID(ctx, eventID)
	if err != nil {
		return 0, fmt.Errorf("ошибка получения события: %w", err)
	}
	// проверяем, что мероприятие ещё не началось
	if event.DateEvent.Before(time.Now()) {
		return 0, fmt.Errorf("мероприятие уже началось или завершилось")
	}
	expiresAt := createdAt.Add(time.Duration(event.BookingTTLMinutes) * time.Minute)

	// 3. Бронируем место в БД
	bookingID, err := s.storage.SeatReserver(ctx, eventID, userID, createdAt, expiresAt)
	if err != nil {
		return 0, fmt.Errorf("ошибка бронирования: %w", err)
	}

	// 4. Добавляем запись в ZSet для отслеживания просрочки
	if err := s.zSet.ZAdd(ctx, float64(expiresAt.Unix()), bookingID); err != nil {
		log.Error("не удалось добавить бронь в ZSet", "id", bookingID, "error", err)
		// компенсация - отменяем созданную бронь в БД
		if cancelErr := s.storage.CancelBooking(ctx, bookingID); cancelErr != nil {
			log.Error("критическая ошибка: не удалось откатить бронь после неудачного добавления в Redis",
				"booking_id", bookingID, "cancel_error", cancelErr)
		}
		return 0, fmt.Errorf("бронь создана, но не зарегистрирована в системе отслеживания: %w", err)
	}

	log.Info("бронь создана", "id", bookingID, "event_id", eventID, "user_id", userID)

	return bookingID, nil
}

// GetEventReserveOfUser - получение данных о брони пользователя на мероприятии (да, один юзер - одно место)
func (s *Service) GetEventReserveOfUser(ctx context.Context, eventID, userID int, log logger.Logger) (int, string, error) {

	bookingID, status, err := s.storage.GetEventReserveOfUser(ctx, eventID, userID)
	if err != nil {
		return 0, "", fmt.Errorf("ошибка получения брони пользователя: %w", err)
	}

	if bookingID == 0 {
		log.Info("бронь не найдена", "event_id", eventID, "user_id", userID)
	} else {
		log.Info("бронь найдена", "id", bookingID)
	}

	return bookingID, status, nil
}

// ReserveConfirmer - метод оплаты/подтверждения бронирования
func (s *Service) ReserveConfirmer(ctx context.Context, bookingID int, log logger.Logger) error {

	if err := s.storage.ReserveConfirmer(ctx, bookingID); err != nil {
		return fmt.Errorf("ошибка подтверждения брони: %w", err)
	}

	// удаляем из ZSet, так как бронь больше не просрочена
	if err := s.zSet.ZRem(ctx, bookingID); err != nil {
		log.Error("не удалось удалить подтверждённую бронь из ZSet", "id", bookingID, "error", err)
	}

	log.Info("бронь подтверждена", "id", bookingID)

	return nil
}

// CancelBooking - отмена брони
func (s *Service) CancelBooking(ctx context.Context, bookingID int, log logger.Logger) error {

	if err := s.storage.CancelBooking(ctx, bookingID); err != nil {
		return fmt.Errorf("ошибка отмены брони в БД: %w", err)
	}

	// удаляем из ZSet, если запись там ещё есть
	if err := s.zSet.ZRem(ctx, bookingID); err != nil {
		log.Error("не удалось удалить отменённую бронь из ZSet", "id", bookingID, "error", err)
	}

	log.Info("бронь отменена", "id", bookingID)

	return nil
}

// RegisterUser - метод для регистрации пользователя
func (s *Service) RegisterUser(ctx context.Context, name, email string, log logger.Logger) (int, error) {

	// проверяем, не занят ли email
	existing, err := s.storage.GetUserByEmail(ctx, email)
	if err != nil {
		return 0, fmt.Errorf("ошибка проверки email: %w", err)
	}

	if existing != 0 {
		return 0, fmt.Errorf("пользователь с таким email уже существует")
	}

	id, err := s.storage.RegisterUser(ctx, name, email)
	if err != nil {
		return 0, fmt.Errorf("ошибка регистрации: %w", err)
	}

	log.Info("пользователь зарегистрирован", "id", id, "email", email)

	return id, nil
}

// LoginUser проверяет наличие пользователя с указанным email и возвращает его ID
func (s *Service) LoginUser(ctx context.Context, email string, log logger.Logger) (int, error) {

	id, err := s.storage.GetUserByEmail(ctx, email)
	if err != nil {
		return 0, fmt.Errorf("ошибка входа: %w", err)
	}

	if id == 0 {
		return 0, nil // пользователь не найден
	}

	log.Info("пользователь вошёл", "id", id)

	return id, nil
}
