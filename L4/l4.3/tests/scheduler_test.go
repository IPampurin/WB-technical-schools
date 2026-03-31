// тесты планировщика напоминаний (Scheduler) с моком EventRepository
package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/IPampurin/EventCalendar/internal/async/scheduler"
	"github.com/IPampurin/EventCalendar/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// мок репозитория для планировщика (только GetPendingReminders)
type mockRepoForScheduler struct {
	pendingFunc func(ctx context.Context, now time.Time) ([]*domain.Event, error)
}

func (m *mockRepoForScheduler) Create(ctx context.Context, e *domain.Event) error { return nil }
func (m *mockRepoForScheduler) Update(ctx context.Context, e *domain.Event) error { return nil }
func (m *mockRepoForScheduler) Delete(ctx context.Context, userID int64, eventID uuid.UUID) error {
	return nil
}
func (m *mockRepoForScheduler) GetByID(ctx context.Context, userID int64, eventID uuid.UUID) (*domain.Event, error) {
	return nil, nil
}
func (m *mockRepoForScheduler) ListBetween(ctx context.Context, userID int64, start, end time.Time) ([]*domain.Event, error) {
	return nil, nil
}
func (m *mockRepoForScheduler) ArchiveOlderThan(ctx context.Context, mark time.Time) (int, error) {
	return 0, nil
}
func (m *mockRepoForScheduler) GetAllArchive(ctx context.Context, userID int64, limit, offset int) ([]*domain.ArchiveEvent, error) {
	return nil, nil
}
func (m *mockRepoForScheduler) GetPendingReminders(ctx context.Context, now time.Time) ([]*domain.Event, error) {
	if m.pendingFunc != nil {
		return m.pendingFunc(ctx, now)
	}
	return nil, nil
}

type mockLoggerSimple struct {
	mu       sync.Mutex
	infoMsgs []string
}

func (l *mockLoggerSimple) Info(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.infoMsgs = append(l.infoMsgs, msg)
}

func (l *mockLoggerSimple) Error(msg string, args ...any) {}
func (l *mockLoggerSimple) Debug(msg string, args ...any) {}
func (l *mockLoggerSimple) Close() error                  { return nil }

func (l *mockLoggerSimple) hasInfo(msg string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, m := range l.infoMsgs {
		if m == msg {
			return true
		}
	}
	return false
}

func TestScheduler_RunAndSchedule(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo := &mockRepoForScheduler{
		pendingFunc: func(ctx context.Context, now time.Time) ([]*domain.Event, error) {
			return nil, nil
		},
	}
	logger := &mockLoggerSimple{}
	sched := scheduler.NewScheduler(ctx, repo, logger, 10)
	go sched.Run()
	defer sched.Stop()

	time.Sleep(50 * time.Millisecond)

	t.Run("Schedule добавляет задачу и она срабатывает", func(t *testing.T) {

		eventID := uuid.New()
		remindAt := time.Now().UTC().Add(150 * time.Millisecond)
		task := domain.ReminderTask{
			EventID:  eventID,
			UserID:   1,
			RemindAt: remindAt,
			Title:    "Test",
		}
		err := sched.Schedule(ctx, task)
		require.NoError(t, err)

		time.Sleep(300 * time.Millisecond)

		assert.True(t, logger.hasInfo("напоминание сработало"))
	})

	t.Run("Cancel отменяет задачу", func(t *testing.T) {

		// Создаём отдельный планировщик для чистоты
		repo2 := &mockRepoForScheduler{pendingFunc: func(ctx context.Context, now time.Time) ([]*domain.Event, error) { return nil, nil }}
		logger2 := &mockLoggerSimple{}
		sched2 := scheduler.NewScheduler(ctx, repo2, logger2, 10)
		go sched2.Run()
		defer sched2.Stop()
		time.Sleep(50 * time.Millisecond)

		eventID := uuid.New()
		remindAt := time.Now().UTC().Add(200 * time.Millisecond)
		task := domain.ReminderTask{EventID: eventID, UserID: 1, RemindAt: remindAt, Title: "Cancel"}
		err := sched2.Schedule(ctx, task)
		require.NoError(t, err)
		err = sched2.Cancel(ctx, eventID)
		require.NoError(t, err)

		time.Sleep(400 * time.Millisecond)
		// Проверяем, что сообщение не появилось
		assert.False(t, logger2.hasInfo("напоминание сработало"), "напоминание не должно было сработать после отмены")
	})
}

func TestScheduler_RestorePending(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	future := time.Now().UTC().Add(150 * time.Millisecond)
	eventID := uuid.New()
	repo := &mockRepoForScheduler{
		pendingFunc: func(ctx context.Context, now time.Time) ([]*domain.Event, error) {
			return []*domain.Event{{ID: eventID, UserID: 1, Title: "Restored", ReminderAt: &future}}, nil
		},
	}
	logger := &mockLoggerSimple{}
	sched := scheduler.NewScheduler(ctx, repo, logger, 10)
	go sched.Run()
	defer sched.Stop()

	time.Sleep(300 * time.Millisecond)

	assert.True(t, logger.hasInfo("напоминание сработало"))
}
