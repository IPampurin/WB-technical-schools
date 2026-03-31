package http

import "github.com/IPampurin/EventCalendar/internal/domain"

// toEventResponse преобразует доменную модель активного события в dto
func toEventResponse(e *domain.Event) EventResponse {

	return EventResponse{
		ID:          e.ID.String(),
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

// toEventResponses преобразует слайс доменных активных событий в слайс dto
func toEventResponses(events []*domain.Event) []EventResponse {

	result := make([]EventResponse, len(events))
	for i, e := range events {
		result[i] = toEventResponse(e)
	}

	return result
}

// toArchiveEventResponse преобразует доменную модель архивного события в dto
func toArchiveEventResponse(e *domain.ArchiveEvent) ArchiveEventResponse {

	return ArchiveEventResponse{
		ID:          e.ID.String(),
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

// toArchiveEventResponses преобразует слайс доменных архивных событий в слайс dto
func toArchiveEventResponses(events []*domain.ArchiveEvent) []ArchiveEventResponse {

	result := make([]ArchiveEventResponse, len(events))
	for i, e := range events {
		result[i] = toArchiveEventResponse(e)
	}

	return result
}
