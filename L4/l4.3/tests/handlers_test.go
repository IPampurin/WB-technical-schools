// tests/handlers_test.go
package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/IPampurin/EventCalendar/internal/domain"
	"github.com/IPampurin/EventCalendar/internal/service"
	calendarhttp "github.com/IPampurin/EventCalendar/internal/transport/http"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Мок для EventRepository (реализует service.EventRepository)
type mockEventRepoForHandler struct {
	createFunc      func(ctx context.Context, e *domain.Event) error
	getByIDFunc     func(ctx context.Context, userID int64, id uuid.UUID) (*domain.Event, error)
	listBetweenFunc func(ctx context.Context, userID int64, start, end time.Time) ([]*domain.Event, error)
}

func (m *mockEventRepoForHandler) Create(ctx context.Context, e *domain.Event) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, e)
	}
	return nil
}
func (m *mockEventRepoForHandler) Update(ctx context.Context, e *domain.Event) error { return nil }
func (m *mockEventRepoForHandler) Delete(ctx context.Context, userID int64, id uuid.UUID) error {
	return nil
}
func (m *mockEventRepoForHandler) GetByID(ctx context.Context, userID int64, id uuid.UUID) (*domain.Event, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, userID, id)
	}
	return nil, domain.ErrNotFound
}
func (m *mockEventRepoForHandler) ListBetween(ctx context.Context, userID int64, start, end time.Time) ([]*domain.Event, error) {
	if m.listBetweenFunc != nil {
		return m.listBetweenFunc(ctx, userID, start, end)
	}
	return nil, nil
}
func (m *mockEventRepoForHandler) ArchiveOlderThan(ctx context.Context, mark time.Time) (int, error) {
	return 0, nil
}
func (m *mockEventRepoForHandler) GetAllArchive(ctx context.Context, userID int64, limit, offset int) ([]*domain.ArchiveEvent, error) {
	return nil, nil
}
func (m *mockEventRepoForHandler) GetPendingReminders(ctx context.Context, now time.Time) ([]*domain.Event, error) {
	return nil, nil
}

// Мок для ReminderScheduler
type mockRemSchedulerForHandler struct {
	scheduleFunc func(ctx context.Context, task domain.ReminderTask) error
	cancelFunc   func(ctx context.Context, id uuid.UUID) error
}

func (m *mockRemSchedulerForHandler) Schedule(ctx context.Context, task domain.ReminderTask) error {
	if m.scheduleFunc != nil {
		return m.scheduleFunc(ctx, task)
	}
	return nil
}
func (m *mockRemSchedulerForHandler) Cancel(ctx context.Context, id uuid.UUID) error {
	if m.cancelFunc != nil {
		return m.cancelFunc(ctx, id)
	}
	return nil
}

// Мок для Logger
type mockLogHandler struct{}

func (l *mockLogHandler) Info(msg string, args ...any)  {}
func (l *mockLogHandler) Error(msg string, args ...any) {}
func (l *mockLogHandler) Debug(msg string, args ...any) {}
func (l *mockLogHandler) Close() error                  { return nil }

func TestHandler_CreateEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		payload    map[string]interface{}
		createErr  error
		wantStatus int
		wantResult string
		wantError  string
	}{
		{
			name: "успешное создание",
			payload: map[string]interface{}{
				"user_id":     1,
				"title":       "Test",
				"start_at":    time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
				"reminder_at": time.Now().UTC().Add(30 * time.Minute).Format(time.RFC3339),
			},
			createErr:  nil,
			wantStatus: http.StatusCreated,
			wantResult: "id",
			wantError:  "",
		},
		{
			name: "неверный JSON",
			payload: map[string]interface{}{
				"user_id": "invalid",
			},
			createErr:  nil,
			wantStatus: http.StatusBadRequest,
			wantResult: "",
			wantError:  "неверный JSON",
		},
		{
			name: "ошибка сервиса",
			payload: map[string]interface{}{
				"user_id":  1,
				"title":    "Test",
				"start_at": time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
			},
			createErr:  errors.New("database error"),
			wantStatus: http.StatusBadRequest,
			wantResult: "",
			wantError:  "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockEventRepoForHandler{
				createFunc: func(ctx context.Context, e *domain.Event) error {
					if tt.createErr != nil {
						return tt.createErr
					}
					e.ID = uuid.New()
					return nil
				},
			}
			sched := &mockRemSchedulerForHandler{}
			svc, _ := service.NewCalendarService(repo, sched, &mockLogHandler{}, "UTC")
			handler := calendarhttp.NewHandler(svc, &mockLogHandler{})

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			body, _ := json.Marshal(tt.payload)
			c.Request = httptest.NewRequest("POST", "/create_event", bytes.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")

			handler.CreateEvent(c)

			assert.Equal(t, tt.wantStatus, w.Code)

			var resp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)

			if tt.wantError != "" {
				assert.Contains(t, resp["error"], tt.wantError)
			} else {
				result, ok := resp["result"].(map[string]interface{})
				assert.True(t, ok)
				assert.Contains(t, result, tt.wantResult)
			}
		})
	}
}

func TestHandler_EventsForMonth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	eventID := uuid.New()

	tests := []struct {
		name       string
		query      string
		serviceErr error
		events     []*domain.Event
		wantStatus int
		wantCount  int
		wantError  string
	}{
		{
			name:       "успешный запрос",
			query:      "?user_id=1&date=2026-03-30",
			serviceErr: nil,
			events:     []*domain.Event{{ID: eventID, Title: "Event"}},
			wantStatus: http.StatusOK,
			wantCount:  1,
			wantError:  "",
		},
		{
			name:       "неверная дата",
			query:      "?user_id=1&date=2026-13-45",
			serviceErr: nil,
			events:     nil,
			wantStatus: http.StatusBadRequest,
			wantCount:  0,
			wantError:  "неверный формат date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockEventRepoForHandler{
				listBetweenFunc: func(ctx context.Context, userID int64, start, end time.Time) ([]*domain.Event, error) {
					return tt.events, tt.serviceErr
				},
			}
			sched := &mockRemSchedulerForHandler{}
			svc, _ := service.NewCalendarService(repo, sched, &mockLogHandler{}, "Europe/Moscow")
			handler := calendarhttp.NewHandler(svc, &mockLogHandler{})

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/events_for_month"+tt.query, nil)

			handler.EventsForMonth(c)

			assert.Equal(t, tt.wantStatus, w.Code)

			var resp map[string]interface{}
			_ = json.Unmarshal(w.Body.Bytes(), &resp)

			if tt.wantError != "" {
				assert.Contains(t, resp["error"], tt.wantError)
			} else {
				result, ok := resp["result"].([]interface{})
				assert.True(t, ok)
				assert.Len(t, result, tt.wantCount)
			}
		})
	}
}
