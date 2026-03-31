// тесты архиватора (Archiver) с моком EventRepository
package tests

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/IPampurin/EventCalendar/internal/async/archiver"
	"github.com/IPampurin/EventCalendar/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// мок для EventRepository
type mockRepoForArchiver struct {
	archiveFunc func(ctx context.Context, mark time.Time) (int, error)
}

func (m *mockRepoForArchiver) Create(ctx context.Context, e *domain.Event) error { return nil }
func (m *mockRepoForArchiver) Update(ctx context.Context, e *domain.Event) error { return nil }
func (m *mockRepoForArchiver) Delete(ctx context.Context, userID int64, eventID uuid.UUID) error {
	return nil
}
func (m *mockRepoForArchiver) GetByID(ctx context.Context, userID int64, eventID uuid.UUID) (*domain.Event, error) {
	return nil, nil
}
func (m *mockRepoForArchiver) ListBetween(ctx context.Context, userID int64, start, end time.Time) ([]*domain.Event, error) {
	return nil, nil
}
func (m *mockRepoForArchiver) ArchiveOlderThan(ctx context.Context, mark time.Time) (int, error) {
	if m.archiveFunc != nil {
		return m.archiveFunc(ctx, mark)
	}
	return 0, nil
}
func (m *mockRepoForArchiver) GetAllArchive(ctx context.Context, userID int64, limit, offset int) ([]*domain.ArchiveEvent, error) {
	return nil, nil
}
func (m *mockRepoForArchiver) GetPendingReminders(ctx context.Context, now time.Time) ([]*domain.Event, error) {
	return nil, nil
}

// потокобезопасный мок-логгер
type mockLoggerForArchiver struct {
	mu     sync.Mutex
	infos  []string
	errors []string
}

func (l *mockLoggerForArchiver) Info(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.infos = append(l.infos, msg)
}
func (l *mockLoggerForArchiver) Error(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errors = append(l.errors, msg)
}
func (l *mockLoggerForArchiver) Debug(msg string, args ...any) {}
func (l *mockLoggerForArchiver) Close() error                  { return nil }

func (l *mockLoggerForArchiver) hasInfo(msg string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, m := range l.infos {
		if m == msg {
			return true
		}
	}
	return false
}

func (l *mockLoggerForArchiver) hasError(msg string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, m := range l.errors {
		if m == msg {
			return true
		}
	}
	return false
}

func TestArchiver_Run(t *testing.T) {
	t.Run("вызывает ArchiveOlderThan по тикеру и логирует успех", func(t *testing.T) {

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		archiveCalled := make(chan struct{})
		repo := &mockRepoForArchiver{
			archiveFunc: func(ctx context.Context, mark time.Time) (int, error) {
				close(archiveCalled)
				return 5, nil
			},
		}
		log := &mockLoggerForArchiver{}

		a := archiver.NewArchiver(ctx, repo, log, 20*time.Millisecond)

		go a.Run()

		select {
		case <-archiveCalled:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("ArchiveOlderThan не был вызван вовремя")
		}

		cancel()
		time.Sleep(30 * time.Millisecond)

		assert.True(t, log.hasInfo("архивация выполнена"))
		assert.True(t, log.hasInfo("архиватор запущен"))
		assert.True(t, log.hasInfo("архиватор остановлен"))
	})

	t.Run("логирует ошибку, если ArchiveOlderThan вернул ошибку", func(t *testing.T) {

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		archiveCalled := make(chan struct{})
		repo := &mockRepoForArchiver{
			archiveFunc: func(ctx context.Context, mark time.Time) (int, error) {
				close(archiveCalled)
				return 0, errors.New("db error")
			},
		}
		log := &mockLoggerForArchiver{}

		a := archiver.NewArchiver(ctx, repo, log, 20*time.Millisecond)

		go a.Run()

		select {
		case <-archiveCalled:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("ArchiveOlderThan не был вызван вовремя")
		}

		cancel()
		time.Sleep(30 * time.Millisecond)

		assert.True(t, log.hasError("ошибка архивации"))
	})

	t.Run("корректно останавливается по отмене контекста", func(t *testing.T) {

		ctx, cancel := context.WithCancel(context.Background())

		repo := &mockRepoForArchiver{}
		log := &mockLoggerForArchiver{}

		a := archiver.NewArchiver(ctx, repo, log, 20*time.Millisecond)

		go a.Run()

		time.Sleep(30 * time.Millisecond)
		cancel()
		time.Sleep(30 * time.Millisecond)

		assert.True(t, log.hasInfo("архиватор остановлен"))
	})

	t.Run("не вызывает ArchiveOlderThan, если интервал не прошёл", func(t *testing.T) {

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var callCount int
		repo := &mockRepoForArchiver{
			archiveFunc: func(ctx context.Context, mark time.Time) (int, error) {
				callCount++
				return 0, nil
			},
		}
		log := &mockLoggerForArchiver{}

		a := archiver.NewArchiver(ctx, repo, log, 1*time.Second)

		go a.Run()

		time.Sleep(100 * time.Millisecond)
		cancel()
		time.Sleep(50 * time.Millisecond)

		assert.Equal(t, 0, callCount, "ArchiveOlderThan вызван раньше времени")
	})
}
