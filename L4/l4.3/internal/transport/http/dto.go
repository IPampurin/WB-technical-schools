// структуры входа-выхода HTTP-слоя
package http

import (
	"time"
)

// входящие запросы (тело JSON, Content-Type: application/json)

// CreateEventRequest - тело POST /create_event
type CreateEventRequest struct {
	UserID      int64   `json:"user_id"`               // id пользователя, к которому относится событие
	Title       string  `json:"title"`                 // название события
	Description string  `json:"description,omitempty"` // описание
	StartAt     string  `json:"start_at"`              // начало события RFC3339 ("2026-03-27T10:00:00+03:00")
	EndAt       *string `json:"end_at,omitempty"`      // окончание события RFC3339 или null (точечное событие)
	ReminderAt  *string `json:"reminder_at,omitempty"` // момент, когда будет отправлено напоминание RFC3339 или null (без напоминания)
}

// UpdateEventRequest - тело POST /update_event
type UpdateEventRequest struct {
	UserID      int64   `json:"user_id"`               // id пользователя
	EventID     string  `json:"event_id"`              // uid события (строка UUID)
	Title       string  `json:"title"`                 // новое название
	Description string  `json:"description,omitempty"` // новое описание
	StartAt     string  `json:"start_at"`              // новое начало: RFC3339
	EndAt       *string `json:"end_at,omitempty"`      // новое окончание или null
	ReminderAt  *string `json:"reminder_at,omitempty"` // новый момент напоминания; null - снять напоминание
}

// DeleteEventRequest - тело POST /delete_event
type DeleteEventRequest struct {
	UserID  int64  `json:"user_id"`  // id пользователя
	EventID string `json:"event_id"` // uid удаляемого события
}

// EventsForPeriodQuery - параметры GET /events_for_day|week|month (query string)
type EventsForPeriodQuery struct {
	UserID int64  `form:"user_id"` // user_id=...
	Date   string `form:"date"`    // date=YYYY-MM-DD - якорная дата для дня/недели/месяца (TZ из конфига)
}

// ArchiveEventsQuery - параметры GET /archive_events
type ArchiveEventsQuery struct {
	UserID int64 `form:"user_id"`          // обязательный id пользователя
	Limit  int   `form:"limit,default=50"` // количество записей на странице
	Offset int   `form:"offset,default=0"` // смещение для пагинации
}

// ответы

// EventResponse - ответ для активного события
type EventResponse struct {
	ID          string     `json:"id"`
	UserID      int64      `json:"user_id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	StartAt     time.Time  `json:"start_at"`
	EndAt       *time.Time `json:"end_at,omitempty"`
	ReminderAt  *time.Time `json:"reminder_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ArchiveEventResponse - ответ для архивного события
type ArchiveEventResponse struct {
	ID          string     `json:"id"`
	UserID      int64      `json:"user_id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	StartAt     time.Time  `json:"start_at"`
	EndAt       *time.Time `json:"end_at,omitempty"`
	ReminderAt  *time.Time `json:"reminder_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ArchivedAt  time.Time  `json:"archived_at"`
}
