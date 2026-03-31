// маппинг полей БД в domain.Event и обратно
package sqldb

import (
	"github.com/IPampurin/EventCalendar/internal/domain"
)

// преобразование доменного события в БД-структуру
func mapEventToDB(e *domain.Event) Event {

	return Event{
		ID:          e.ID,
		UserID:      e.UserID,
		Title:       e.Title,
		Description: e.Description,
		StartAt:     e.StartAt,
		EndAt:       e.EndAt,
		ReminderAt:  e.ReminderAt,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

// преобразование БД-структуры в доменное событие
func mapDBToEvent(d Event) *domain.Event {

	return &domain.Event{
		ID:          d.ID,
		UserID:      d.UserID,
		Title:       d.Title,
		Description: d.Description,
		StartAt:     d.StartAt,
		EndAt:       d.EndAt,
		ReminderAt:  d.ReminderAt,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}

// преобразование доменного архивного события в БД-структуру
func mapArchiveEventToDB(e *domain.ArchiveEvent) ArchiveEvent {

	return ArchiveEvent{
		ID:          e.ID,
		UserID:      e.UserID,
		Title:       e.Title,
		Description: e.Description,
		StartAt:     e.StartAt,
		EndAt:       e.EndAt,
		ReminderAt:  e.ReminderAt,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
		ArchivedAt:  e.ArchivedAt,
	}
}

// преобразование БД-структуры в доменное архивное событие
func mapDBToArchiveEvent(d ArchiveEvent) *domain.ArchiveEvent {

	return &domain.ArchiveEvent{
		ID:          d.ID,
		UserID:      d.UserID,
		Title:       d.Title,
		Description: d.Description,
		StartAt:     d.StartAt,
		EndAt:       d.EndAt,
		ReminderAt:  d.ReminderAt,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
		ArchivedAt:  d.ArchivedAt,
	}
}
