// интерфейс напоминальщика - порт планировщика (канал, брокер и т.д.)
package service

import (
	"context"

	"github.com/IPampurin/EventCalendar/internal/domain"
	"github.com/google/uuid"
)

// ReminderScheduler - постановка и отмена напоминаний
type ReminderScheduler interface {

	// Schedule - поставить напоминание в очередь (асинхронно)
	Schedule(ctx context.Context, task domain.ReminderTask) error

	// Cancel - снять напоминание для события (при удалении события или снятии reminder)
	Cancel(ctx context.Context, eventID uuid.UUID) error
}
