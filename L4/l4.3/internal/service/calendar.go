// логика приложения
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/IPampurin/EventCalendar/internal/domain"
	"github.com/google/uuid"
)

type CalendarService struct {
	repo EventRepository
	rem  ReminderScheduler
	log  Logger
	tz   *time.Location // часовой пояс для расчётов дня/недели/месяца
}

func NewCalendarService(repo EventRepository, rem ReminderScheduler, logger Logger, tzName string) (*CalendarService, error) {

	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return nil, fmt.Errorf("неверный часовой пояс: %w", err)
	}

	return &CalendarService{
		repo: repo,
		rem:  rem,
		log:  logger,
		tz:   loc,
	}, nil
}

// Create создаёт новое событие
func (s *CalendarService) Create(ctx context.Context, event *domain.Event) error {

	// валидация
	if err := validateEvent(event); err != nil {
		return err
	}

	// установка временных меток
	now := time.Now().UTC()
	event.ID = uuid.New()
	event.CreatedAt = now
	event.UpdatedAt = now

	if err := s.repo.Create(ctx, event); err != nil {
		return fmt.Errorf("ошибка сохранения события: %w", err)
	}

	// если задано напоминание - ставим задачу в планировщик
	if event.ReminderAt != nil {
		task := domain.ReminderTask{
			EventID:  event.ID,
			UserID:   event.UserID,
			RemindAt: *event.ReminderAt,
			Title:    event.Title,
		}
		if err := s.rem.Schedule(ctx, task); err != nil {
			s.log.Error("не удалось запланировать напоминание", "event_id", event.ID, "error", err)
			// ошибка планирования не должна отменять создание события
		}
	}

	s.log.Info("событие создано", "event_id", event.ID, "user_id", event.UserID)

	return nil
}

// Update обновляет существующее событие
func (s *CalendarService) Update(ctx context.Context, event *domain.Event) error {

	if err := validateEvent(event); err != nil {
		return err
	}

	// получаем старое событие, чтобы знать, нужно ли перепланировать напоминание
	old, err := s.repo.GetByID(ctx, event.UserID, event.ID)
	if err != nil {
		return fmt.Errorf("не удалось получить событие для обновления: %w", err)
	}

	event.UpdatedAt = time.Now().UTC()
	// не меняем CreatedAt и ID
	if err := s.repo.Update(ctx, event); err != nil {
		return err
	}

	// управление напоминаниями
	if needReschedule(old, event) {
		// отменяем старое
		if old.ReminderAt != nil {
			_ = s.rem.Cancel(ctx, event.ID)
		}
		// планируем новое, если есть
		if event.ReminderAt != nil {
			task := domain.ReminderTask{
				EventID:  event.ID,
				UserID:   event.UserID,
				RemindAt: *event.ReminderAt,
				Title:    event.Title,
			}
			if err := s.rem.Schedule(ctx, task); err != nil {
				s.log.Error("не удалось перепланировать напоминание", "event_id", event.ID, "error", err)
			}
		}
	}

	s.log.Info("событие обновлено", "event_id", event.ID, "user_id", event.UserID)

	return nil
}

// Delete удаляет событие
func (s *CalendarService) Delete(ctx context.Context, userID int64, eventID uuid.UUID) error {

	// перед удалением отменяем напоминание, если было
	event, err := s.repo.GetByID(ctx, userID, eventID)
	if err == nil && event.ReminderAt != nil {
		_ = s.rem.Cancel(ctx, eventID)
	}

	if err := s.repo.Delete(ctx, userID, eventID); err != nil {
		return err
	}

	s.log.Info("событие удалено", "event_id", eventID, "user_id", userID)

	return nil
}

// GetByID возвращает событие по ID (только активные)
func (s *CalendarService) GetByID(ctx context.Context, userID int64, eventID uuid.UUID) (*domain.Event, error) {
	return s.repo.GetByID(ctx, userID, eventID)
}

// GetEventsForDay возвращает события на календарный день (в указанной временной зоне)
func (s *CalendarService) GetEventsForDay(ctx context.Context, userID int64, date time.Time) ([]*domain.Event, error) {

	// date - это дата, переданная клиентом в формате YYYY-MM-DD, интерпретируется как локальная дата в tz сервера
	localDate := date.In(s.tz)
	start := time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 0, 0, 0, 0, s.tz)
	end := start.AddDate(0, 0, 1)

	return s.repo.ListBetween(ctx, userID, start.UTC(), end.UTC())
}

// GetEventsForWeek возвращает события на неделю, начинающуюся с даты (понедельник–воскресенье)
func (s *CalendarService) GetEventsForWeek(ctx context.Context, userID int64, date time.Time) ([]*domain.Event, error) {

	localDate := date.In(s.tz)
	// находим понедельник недели
	weekday := localDate.Weekday()
	offset := int(weekday - time.Monday)
	if offset < 0 {
		offset += 7
	}

	start := time.Date(localDate.Year(), localDate.Month(), localDate.Day()-offset, 0, 0, 0, 0, s.tz)
	end := start.AddDate(0, 0, 7)

	return s.repo.ListBetween(ctx, userID, start.UTC(), end.UTC())
}

// GetEventsForMonth возвращает события на календарный месяц
func (s *CalendarService) GetEventsForMonth(ctx context.Context, userID int64, date time.Time) ([]*domain.Event, error) {

	localDate := date.In(s.tz)
	start := time.Date(localDate.Year(), localDate.Month(), 1, 0, 0, 0, 0, s.tz)
	end := start.AddDate(0, 1, 0)

	return s.repo.ListBetween(ctx, userID, start.UTC(), end.UTC())
}

// GetArchiveEvents возвращает архивные события пользователя с пагинацией
func (s *CalendarService) GetArchiveEvents(ctx context.Context, userID int64, limit, offset int) ([]*domain.ArchiveEvent, error) {

	// предполагаем, что у репозитория есть метод GetAllArchive
	return s.repo.GetAllArchive(ctx, userID, limit, offset)
}
