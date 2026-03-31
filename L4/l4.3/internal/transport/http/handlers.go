// обработчики HTTP
package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/IPampurin/EventCalendar/internal/domain"
	"github.com/IPampurin/EventCalendar/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler обрабатывает HTTP-запросы для API календаря событий
type Handler struct {
	svc    *service.CalendarService // предоставляет методы для работы с событиями календаря
	logger service.Logger           // используется для асинхронного логирования операций
}

// NewHandler возвращает сконфигурированный Handler
func NewHandler(svc *service.CalendarService, logger service.Logger) *Handler {

	return &Handler{
		svc:    svc,    // инъекция сервиса для делегирования бизнес-операций
		logger: logger, // инъекция логгера для асинхронной записи логов через канал
	}
}

// ошибка с соответствующим статусом
func respondError(c *gin.Context, err error) {

	var status int
	var message string

	switch {
	case errors.Is(err, domain.ErrNotFound):
		status = http.StatusNotFound // 404
		message = "событие не найдено"
	case errors.Is(err, context.DeadlineExceeded):
		status = http.StatusGatewayTimeout // 504
		message = "таймаут операции"
	default:
		// для всех остальных ошибок (в т.ч. валидации) - 400
		status = http.StatusBadRequest
		message = err.Error()
	}

	c.JSON(status, gin.H{"error": message})
}

// CreateEvent POST /create_event
func (h *Handler) CreateEvent(c *gin.Context) {

	var req CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, fmt.Errorf("неверный JSON: %w", err))
		return
	}

	event, err := req.ToDomain()
	if err != nil {
		respondError(c, err)
		return
	}

	if err := h.svc.Create(c.Request.Context(), event); err != nil {
		respondError(c, err)
		return
	}

	// 201 Created — ресурс создан
	c.JSON(http.StatusCreated, gin.H{"result": gin.H{"id": event.ID.String()}})
}

// UpdateEvent POST /update_event
func (h *Handler) UpdateEvent(c *gin.Context) {

	var req UpdateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, fmt.Errorf("неверный JSON: %w", err))
		return
	}

	event, err := req.ToDomain()
	if err != nil {
		respondError(c, err)
		return
	}

	if err := h.svc.Update(c.Request.Context(), event); err != nil {
		respondError(c, err)
		return
	}

	// 200 OK — обновление успешно
	c.JSON(http.StatusOK, gin.H{"result": gin.H{"id": event.ID.String()}})
}

// DeleteEvent POST /delete_event
func (h *Handler) DeleteEvent(c *gin.Context) {

	var req DeleteEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, fmt.Errorf("неверный JSON: %w", err))
		return
	}

	eventID, err := uuid.Parse(req.EventID)
	if err != nil {
		respondError(c, fmt.Errorf("неверный event_id: %w", err))
		return
	}

	if err := h.svc.Delete(c.Request.Context(), req.UserID, eventID); err != nil {
		respondError(c, err)
		return
	}

	// 200 OK — удаление успешно
	c.JSON(http.StatusOK, gin.H{"result": gin.H{"status": "deleted"}})
}

// EventsForDay GET /events_for_day
func (h *Handler) EventsForDay(c *gin.Context) {

	var query EventsForPeriodQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		respondError(c, fmt.Errorf("неверные параметры запроса: %w", err))
		return
	}

	date, err := time.Parse("2006-01-02", query.Date)
	if err != nil {
		respondError(c, fmt.Errorf("неверный формат date (требуется YYYY-MM-DD): %w", err))
		return
	}

	events, err := h.svc.GetEventsForDay(c.Request.Context(), query.UserID, date)
	if err != nil {
		respondError(c, err)
		return
	}

	// 200 OK — список событий
	c.JSON(http.StatusOK, gin.H{"result": toEventResponses(events)})
}

// EventsForWeek GET /events_for_week
func (h *Handler) EventsForWeek(c *gin.Context) {

	var query EventsForPeriodQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		respondError(c, fmt.Errorf("неверные параметры запроса: %w", err))
		return
	}

	date, err := time.Parse("2006-01-02", query.Date)
	if err != nil {
		respondError(c, fmt.Errorf("неверный формат date (требуется YYYY-MM-DD): %w", err))
		return
	}

	events, err := h.svc.GetEventsForWeek(c.Request.Context(), query.UserID, date)
	if err != nil {
		respondError(c, err)
		return
	}

	// 200 OK — список событий
	c.JSON(http.StatusOK, gin.H{"result": toEventResponses(events)})
}

// EventsForMonth GET /events_for_month
func (h *Handler) EventsForMonth(c *gin.Context) {

	var query EventsForPeriodQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		respondError(c, fmt.Errorf("неверные параметры запроса: %w", err))
		return
	}

	date, err := time.Parse("2006-01-02", query.Date)
	if err != nil {
		respondError(c, fmt.Errorf("неверный формат date (требуется YYYY-MM-DD): %w", err))
		return
	}

	events, err := h.svc.GetEventsForMonth(c.Request.Context(), query.UserID, date)
	if err != nil {
		respondError(c, err)
		return
	}

	// 200 OK — список событий
	c.JSON(http.StatusOK, gin.H{"result": toEventResponses(events)})
}

// GetArchiveEvents GET /archive_events
func (h *Handler) GetArchiveEvents(c *gin.Context) {

	var query ArchiveEventsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		respondError(c, fmt.Errorf("неверные параметры запроса: %w", err))
		return
	}
	if query.UserID == 0 {
		respondError(c, fmt.Errorf("user_id обязателен"))
		return
	}
	if query.Limit <= 0 {
		query.Limit = 50
	}
	if query.Offset < 0 {
		query.Offset = 0
	}

	events, err := h.svc.GetArchiveEvents(c.Request.Context(), query.UserID, query.Limit, query.Offset)
	if err != nil {
		respondError(c, err)
		return
	}

	// 200 OK — список архивных событий
	c.JSON(http.StatusOK, gin.H{"result": toArchiveEventResponses(events)})
}

// ToDomain создаёт domain.Event из CreateEventRequest
func (r CreateEventRequest) ToDomain() (*domain.Event, error) {

	startAt, err := time.Parse(time.RFC3339, r.StartAt)
	if err != nil {
		return nil, fmt.Errorf("неверный формат start_at (требуется RFC3339): %w", err)
	}

	var endAt *time.Time
	if r.EndAt != nil && *r.EndAt != "" {
		t, err := time.Parse(time.RFC3339, *r.EndAt)
		if err != nil {
			return nil, fmt.Errorf("неверный формат end_at: %w", err)
		}
		endAt = &t
	}

	var reminderAt *time.Time
	if r.ReminderAt != nil && *r.ReminderAt != "" {
		t, err := time.Parse(time.RFC3339, *r.ReminderAt)
		if err != nil {
			return nil, fmt.Errorf("неверный формат reminder_at: %w", err)
		}
		reminderAt = &t
	}

	return &domain.Event{
		UserID:      r.UserID,
		Title:       r.Title,
		Description: r.Description,
		StartAt:     startAt.UTC(),
		EndAt:       endAt,
		ReminderAt:  reminderAt,
	}, nil
}

// ToDomain для UpdateEventRequest
func (r UpdateEventRequest) ToDomain() (*domain.Event, error) {

	eventID, err := uuid.Parse(r.EventID)
	if err != nil {
		return nil, fmt.Errorf("неверный event_id: %w", err)
	}

	startAt, err := time.Parse(time.RFC3339, r.StartAt)
	if err != nil {
		return nil, fmt.Errorf("неверный формат start_at: %w", err)
	}

	var endAt *time.Time
	if r.EndAt != nil && *r.EndAt != "" {
		t, err := time.Parse(time.RFC3339, *r.EndAt)
		if err != nil {
			return nil, fmt.Errorf("неверный формат end_at: %w", err)
		}
		endAt = &t
	}

	var reminderAt *time.Time
	if r.ReminderAt != nil && *r.ReminderAt != "" {
		t, err := time.Parse(time.RFC3339, *r.ReminderAt)
		if err != nil {
			return nil, fmt.Errorf("неверный формат reminder_at: %w", err)
		}
		reminderAt = &t
	}

	return &domain.Event{
		ID:          eventID,
		UserID:      r.UserID,
		Title:       r.Title,
		Description: r.Description,
		StartAt:     startAt.UTC(),
		EndAt:       endAt,
		ReminderAt:  reminderAt,
	}, nil
}
