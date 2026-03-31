// функции валидации
package service

import (
	"fmt"
	"time"

	"github.com/IPampurin/EventCalendar/internal/domain"
)

// validateEvent выполняет базовую валидацию полей события
func validateEvent(e *domain.Event) error {

	if e.UserID == 0 {
		return fmt.Errorf("user_id обязателен")
	}
	if e.Title == "" {
		return fmt.Errorf("title не может быть пустым")
	}
	if e.StartAt.IsZero() {
		return fmt.Errorf("start_at обязателен")
	}

	// проверка хронологической последовательности ReminderAt -> StartAt -> EndAt
	if e.ReminderAt != nil {
		if e.ReminderAt.Before(time.Now().UTC()) {
			return fmt.Errorf("reminder_at не может быть в прошлом")
		}
		if !e.ReminderAt.Before(e.StartAt) {
			return fmt.Errorf("reminder_at должен быть раньше start_at")
		}
	}
	if e.EndAt != nil && e.EndAt.Before(e.StartAt) {
		return fmt.Errorf("end_at не может быть раньше start_at")
	}

	return nil
}

// needReschedule определяет, нужно ли перепланировать напоминание
func needReschedule(old, new *domain.Event) bool {

	// если напоминания совпадают по времени - можно не перепланировать
	if old.ReminderAt == nil && new.ReminderAt == nil {
		return false
	}
	if old.ReminderAt == nil && new.ReminderAt != nil {
		return true
	}
	if old.ReminderAt != nil && new.ReminderAt == nil {
		return true
	}

	// оба не nil
	return !old.ReminderAt.Equal(*new.ReminderAt)
}
