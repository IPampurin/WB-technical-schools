// тесты бизнес-логики (CalendarService) с моком EventRepository и ReminderScheduler
package tests

import (
	"context"
	"testing"
	"time"

	"github.com/IPampurin/EventCalendar/internal/domain"
	"github.com/IPampurin/EventCalendar/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// мок для EventRepository
type mockEventRepository struct {
	createFunc      func(ctx context.Context, e *domain.Event) error
	updateFunc      func(ctx context.Context, e *domain.Event) error
	deleteFunc      func(ctx context.Context, userID int64, eventID uuid.UUID) error
	getByIDFunc     func(ctx context.Context, userID int64, eventID uuid.UUID) (*domain.Event, error)
	listBetweenFunc func(ctx context.Context, userID int64, start, end time.Time) ([]*domain.Event, error)
}

func (m *mockEventRepository) Create(ctx context.Context, e *domain.Event) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, e)
	}
	return nil
}

func (m *mockEventRepository) Update(ctx context.Context, e *domain.Event) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, e)
	}
	return nil
}
func (m *mockEventRepository) Delete(ctx context.Context, userID int64, eventID uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, userID, eventID)
	}
	return nil
}
func (m *mockEventRepository) GetByID(ctx context.Context, userID int64, eventID uuid.UUID) (*domain.Event, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, userID, eventID)
	}
	return nil, domain.ErrNotFound
}
func (m *mockEventRepository) ListBetween(ctx context.Context, userID int64, start, end time.Time) ([]*domain.Event, error) {
	if m.listBetweenFunc != nil {
		return m.listBetweenFunc(ctx, userID, start, end)
	}
	return nil, nil
}
func (m *mockEventRepository) ArchiveOlderThan(ctx context.Context, mark time.Time) (int, error) {
	return 0, nil
}
func (m *mockEventRepository) GetAllArchive(ctx context.Context, userID int64, limit, offset int) ([]*domain.ArchiveEvent, error) {
	return nil, nil
}
func (m *mockEventRepository) GetPendingReminders(ctx context.Context, now time.Time) ([]*domain.Event, error) {
	return nil, nil
}

// мок для ReminderScheduler
type mockReminderScheduler struct {
	scheduleFunc func(ctx context.Context, task domain.ReminderTask) error
	cancelFunc   func(ctx context.Context, eventID uuid.UUID) error
}

func (m *mockReminderScheduler) Schedule(ctx context.Context, task domain.ReminderTask) error {
	if m.scheduleFunc != nil {
		return m.scheduleFunc(ctx, task)
	}
	return nil
}
func (m *mockReminderScheduler) Cancel(ctx context.Context, eventID uuid.UUID) error {
	if m.cancelFunc != nil {
		return m.cancelFunc(ctx, eventID)
	}
	return nil
}

// мок для Logger
type mockLogger struct{}

func (l *mockLogger) Info(msg string, args ...any)  {}
func (l *mockLogger) Error(msg string, args ...any) {}
func (l *mockLogger) Debug(msg string, args ...any) {}
func (l *mockLogger) Close() error                  { return nil }

func TestCalendarService_Create(t *testing.T) {

	ctx := context.Background()

	t.Run("успешное создание события без напоминания", func(t *testing.T) {
		repo := &mockEventRepository{
			createFunc: func(ctx context.Context, e *domain.Event) error {
				assert.NotEmpty(t, e.ID)
				assert.NotZero(t, e.CreatedAt)
				return nil
			},
		}
		sched := &mockReminderScheduler{}
		svc, _ := service.NewCalendarService(repo, sched, &mockLogger{}, "UTC")
		event := &domain.Event{
			UserID:      1,
			Title:       "Test Event",
			Description: "Description",
			StartAt:     time.Now().UTC().Add(time.Hour),
		}
		err := svc.Create(ctx, event)
		require.NoError(t, err)
	})

	t.Run("успешное создание с напоминанием - планировщик вызывается", func(t *testing.T) {

		var scheduled bool
		repo := &mockEventRepository{createFunc: func(ctx context.Context, e *domain.Event) error { return nil }}
		sched := &mockReminderScheduler{
			scheduleFunc: func(ctx context.Context, task domain.ReminderTask) error {
				scheduled = true
				return nil
			},
		}
		svc, _ := service.NewCalendarService(repo, sched, &mockLogger{}, "UTC")
		reminderAt := time.Now().UTC().Add(time.Hour)
		event := &domain.Event{
			UserID:     1,
			Title:      "Test",
			StartAt:    time.Now().UTC().Add(2 * time.Hour),
			ReminderAt: &reminderAt,
		}
		err := svc.Create(ctx, event)
		require.NoError(t, err)
		assert.True(t, scheduled)
	})

	t.Run("ошибка валидации", func(t *testing.T) {

		svc, _ := service.NewCalendarService(&mockEventRepository{}, &mockReminderScheduler{}, &mockLogger{}, "UTC")
		event := &domain.Event{UserID: 0, Title: ""}
		err := svc.Create(ctx, event)
		assert.Error(t, err)
	})
}

func TestCalendarService_GetEventsForMonth(t *testing.T) {

	ctx := context.Background()
	date := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)

	t.Run("возвращает события за месяц", func(t *testing.T) {

		expectedEvents := []*domain.Event{{ID: uuid.New(), Title: "Event1"}}
		repo := &mockEventRepository{
			listBetweenFunc: func(ctx context.Context, userID int64, start, end time.Time) ([]*domain.Event, error) {
				assert.Equal(t, int64(1), userID)
				return expectedEvents, nil
			},
		}
		svc, _ := service.NewCalendarService(repo, &mockReminderScheduler{}, &mockLogger{}, "Europe/Moscow")
		events, err := svc.GetEventsForMonth(ctx, 1, date)
		require.NoError(t, err)
		assert.Len(t, events, 1)
	})
}
