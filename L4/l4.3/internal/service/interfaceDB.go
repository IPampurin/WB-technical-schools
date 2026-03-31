// интерфейс БД - порт доступа к событиям в хранилище
package service

import (
	"context"
	"time"

	"github.com/IPampurin/EventCalendar/internal/domain"
	"github.com/google/uuid"
)

// EventRepository - чтение и изменение событий календаря
type EventRepository interface {

	// Create - вставка новой записи
	Create(ctx context.Context, e *domain.Event) error

	// Update - обновление существующего события по ID (ожидается проверка владельца через UserID в e)
	Update(ctx context.Context, e *domain.Event) error

	// Delete - удаление по событию и пользователю (ошибка если не найдено или не совпал user)
	Delete(ctx context.Context, userID int64, eventID uuid.UUID) error

	// GetByID - одно событие (nil если нет или не тот user)
	GetByID(ctx context.Context, userID int64, eventID uuid.UUID) (*domain.Event, error)

	// ListBetween - события пользователя, у которых интервал пересекается с [start, end) в смысле хранения (UTC)
	// (границы day/week/month считает сервис и передаёт start/end сюда)
	ListBetween(ctx context.Context, userID int64, start, end time.Time) ([]*domain.Event, error)

	// ArchiveOlderThan - переносит прошедшие (StartAt, при EndAt == nil) и закончившиеся (EndAt) события в архив
	ArchiveOlderThan(ctx context.Context, mark time.Time) (archived int, err error)

	// GetAllArchive получение всех архивных событий (с пагинацией)
	GetAllArchive(ctx context.Context, userID int64, limit, offset int) ([]*domain.ArchiveEvent, error)

	// GetPendingReminders возвращает события с ReminderAt > now (для восстановления напоминаний при старте)
	GetPendingReminders(ctx context.Context, now time.Time) ([]*domain.Event, error)
}
