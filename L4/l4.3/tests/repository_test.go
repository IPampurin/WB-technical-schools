// тесты репозитория (sqlDB) с реальной PostgreSQL (testcontainers)
package tests

import (
	"context"
	"testing"
	"time"

	"github.com/IPampurin/EventCalendar/internal/domain"
	sqlDB "github.com/IPampurin/EventCalendar/internal/storage/sqlDB"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var testDB *pgxpool.Pool
var testStore *sqlDB.Store

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Запускаем контейнер PostgreSQL
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(30 * time.Second),
	}

	postgresContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		panic(err)
	}
	defer func() { _ = postgresContainer.Terminate(ctx) }()

	host, err := postgresContainer.Host(ctx)
	if err != nil {
		panic(err)
	}
	port, err := postgresContainer.MappedPort(ctx, "5432")
	if err != nil {
		panic(err)
	}

	// Формируем DSN
	dsn := "postgres://test:test@" + host + ":" + port.Port() + "/testdb?sslmode=disable"

	// Подключаемся
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	// Применяем миграции
	if err := sqlDB.Migrations(ctx, pool); err != nil {
		panic(err)
	}

	testDB = pool
	testStore = sqlDB.NewStore(pool)

	// Запускаем тесты
	m.Run()
}

// cleanupTables очищает таблицы после каждого теста
func cleanupTables(t *testing.T) {
	_, err := testDB.Exec(context.Background(), "TRUNCATE TABLE archive_events, events RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}

// Тесты

func TestStore_Create(t *testing.T) {
	cleanupTables(t)

	ctx := context.Background()
	event := &domain.Event{
		ID:          uuid.New(),
		UserID:      123,
		Title:       "Test Event",
		Description: "Description",
		StartAt:     time.Now().UTC().Add(time.Hour),
		EndAt:       nil,
		ReminderAt:  nil,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	err := testStore.Create(ctx, event)
	assert.NoError(t, err)

	// Проверяем, что событие существует
	got, err := testStore.GetByID(ctx, event.UserID, event.ID)
	assert.NoError(t, err)
	assert.Equal(t, event.Title, got.Title)
	assert.Equal(t, event.Description, got.Description)
	assert.WithinDuration(t, event.StartAt, got.StartAt, time.Second)
}

func TestStore_Update(t *testing.T) {
	cleanupTables(t)

	ctx := context.Background()
	event := &domain.Event{
		ID:          uuid.New(),
		UserID:      123,
		Title:       "Old Title",
		Description: "Old Desc",
		StartAt:     time.Now().UTC().Add(time.Hour),
		EndAt:       nil,
		ReminderAt:  nil,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	err := testStore.Create(ctx, event)
	require.NoError(t, err)

	// Обновляем
	event.Title = "New Title"
	event.Description = "New Desc"
	newStart := time.Now().UTC().Add(2 * time.Hour)
	event.StartAt = newStart
	err = testStore.Update(ctx, event)
	assert.NoError(t, err)

	// Проверяем
	got, err := testStore.GetByID(ctx, event.UserID, event.ID)
	assert.NoError(t, err)
	assert.Equal(t, "New Title", got.Title)
	assert.Equal(t, "New Desc", got.Description)
	assert.WithinDuration(t, newStart, got.StartAt, time.Second)
}

func TestStore_Delete(t *testing.T) {
	cleanupTables(t)

	ctx := context.Background()
	event := &domain.Event{
		ID:        uuid.New(),
		UserID:    123,
		Title:     "To Delete",
		StartAt:   time.Now().UTC().Add(time.Hour),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	err := testStore.Create(ctx, event)
	require.NoError(t, err)

	err = testStore.Delete(ctx, event.UserID, event.ID)
	assert.NoError(t, err)

	_, err = testStore.GetByID(ctx, event.UserID, event.ID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestStore_GetByID_NotFound(t *testing.T) {
	cleanupTables(t)

	ctx := context.Background()
	_, err := testStore.GetByID(ctx, 999, uuid.New())
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestStore_ListBetween(t *testing.T) {
	cleanupTables(t)

	ctx := context.Background()
	userID := int64(1)
	now := time.Now().UTC()

	// Событие, которое пересекает интервал
	event1 := &domain.Event{
		ID:        uuid.New(),
		UserID:    userID,
		Title:     "Event1",
		StartAt:   now.Add(-1 * time.Hour),
		EndAt:     nil,
		CreatedAt: now,
		UpdatedAt: now,
	}
	// Событие, которое начинается до интервала и заканчивается внутри
	event2 := &domain.Event{
		ID:        uuid.New(),
		UserID:    userID,
		Title:     "Event2",
		StartAt:   now.Add(-2 * time.Hour),
		EndAt:     ptrTime(now.Add(30 * time.Minute)),
		CreatedAt: now,
		UpdatedAt: now,
	}
	// Событие после интервала (не должно попасть)
	event3 := &domain.Event{
		ID:        uuid.New(),
		UserID:    userID,
		Title:     "Event3",
		StartAt:   now.Add(2 * time.Hour),
		EndAt:     nil,
		CreatedAt: now,
		UpdatedAt: now,
	}
	// Событие другого пользователя
	event4 := &domain.Event{
		ID:        uuid.New(),
		UserID:    2,
		Title:     "Other",
		StartAt:   now.Add(-1 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}

	for _, e := range []*domain.Event{event1, event2, event3, event4} {
		err := testStore.Create(ctx, e)
		require.NoError(t, err)
	}

	// Интервал: [now-1h, now+1h)
	start := now.Add(-1 * time.Hour)
	end := now.Add(1 * time.Hour)

	events, err := testStore.ListBetween(ctx, userID, start, end)
	assert.NoError(t, err)
	assert.Len(t, events, 2) // event1 и event2

	// Проверяем, что event1 и event2 присутствуют
	titles := make([]string, len(events))
	for i, e := range events {
		titles[i] = e.Title
	}
	assert.Contains(t, titles, "Event1")
	assert.Contains(t, titles, "Event2")
	assert.NotContains(t, titles, "Event3")
	assert.NotContains(t, titles, "Other")
}

func TestStore_ArchiveOlderThan(t *testing.T) {
	cleanupTables(t)

	ctx := context.Background()
	now := time.Now().UTC()

	// Событие, которое уже завершилось (EndAt < now)
	event1 := &domain.Event{
		ID:        uuid.New(),
		UserID:    1,
		Title:     "Finished",
		StartAt:   now.Add(-2 * time.Hour),
		EndAt:     ptrTime(now.Add(-1 * time.Hour)),
		CreatedAt: now,
		UpdatedAt: now,
	}
	// Событие без EndAt, но StartAt < now (должно быть заархивировано)
	event2 := &domain.Event{
		ID:        uuid.New(),
		UserID:    1,
		Title:     "Started and finished",
		StartAt:   now.Add(-30 * time.Minute),
		EndAt:     nil,
		CreatedAt: now,
		UpdatedAt: now,
	}
	// Событие будущее (не должно архивироваться)
	event3 := &domain.Event{
		ID:        uuid.New(),
		UserID:    1,
		Title:     "Future",
		StartAt:   now.Add(2 * time.Hour),
		EndAt:     nil,
		CreatedAt: now,
		UpdatedAt: now,
	}
	// Событие с EndAt в будущем (не должно)
	event4 := &domain.Event{
		ID:        uuid.New(),
		UserID:    1,
		Title:     "Ends future",
		StartAt:   now.Add(-1 * time.Hour),
		EndAt:     ptrTime(now.Add(1 * time.Hour)),
		CreatedAt: now,
		UpdatedAt: now,
	}

	for _, e := range []*domain.Event{event1, event2, event3, event4} {
		err := testStore.Create(ctx, e)
		require.NoError(t, err)
	}

	// Архивация: mark = now
	archived, err := testStore.ArchiveOlderThan(ctx, now)
	assert.NoError(t, err)
	assert.Equal(t, 2, archived, "Должны быть заархивированы event1 и event2")

	// Проверяем, что заархивированные события удалены из активных
	_, err = testStore.GetByID(ctx, 1, event1.ID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	_, err = testStore.GetByID(ctx, 1, event2.ID)
	assert.ErrorIs(t, err, domain.ErrNotFound)

	// Проверяем, что будущие остались
	_, err = testStore.GetByID(ctx, 1, event3.ID)
	assert.NoError(t, err)
	_, err = testStore.GetByID(ctx, 1, event4.ID)
	assert.NoError(t, err)

	// Проверяем архивные записи
	archives, err := testStore.GetAllArchive(ctx, 1, 100, 0)
	assert.NoError(t, err)
	assert.Len(t, archives, 2)
	titles := []string{archives[0].Title, archives[1].Title}
	assert.Contains(t, titles, "Finished")
	assert.Contains(t, titles, "Started and finished")
}

func TestStore_GetAllArchive(t *testing.T) {
	cleanupTables(t)

	ctx := context.Background()
	userID := int64(1)

	// Создаём несколько архивных записей напрямую (через метод ArchiveEventCreate)
	for i := 0; i < 5; i++ {
		arch := &domain.ArchiveEvent{
			ID:          uuid.New(),
			UserID:      userID,
			Title:       "Archived " + string(rune('A'+i)),
			Description: "",
			StartAt:     time.Now().UTC().Add(-time.Hour),
			EndAt:       nil,
			ReminderAt:  nil,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
			ArchivedAt:  time.Now().UTC(),
		}
		err := testStore.ArchiveEventCreate(ctx, arch)
		require.NoError(t, err)
	}

	// Проверяем пагинацию
	page1, err := testStore.GetAllArchive(ctx, userID, 2, 0)
	assert.NoError(t, err)
	assert.Len(t, page1, 2)

	page2, err := testStore.GetAllArchive(ctx, userID, 2, 2)
	assert.NoError(t, err)
	assert.Len(t, page2, 2)

	all, err := testStore.GetAllArchive(ctx, userID, 100, 0)
	assert.NoError(t, err)
	assert.Len(t, all, 5)
}

func TestStore_GetPendingReminders(t *testing.T) {
	cleanupTables(t)

	ctx := context.Background()
	now := time.Now().UTC()

	// Напоминание в будущем
	future := now.Add(1 * time.Hour)
	eventFuture := &domain.Event{
		ID:         uuid.New(),
		UserID:     1,
		Title:      "Future reminder",
		StartAt:    now.Add(2 * time.Hour),
		ReminderAt: &future,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	// Напоминание в прошлом (не должно быть получено)
	past := now.Add(-1 * time.Hour)
	eventPast := &domain.Event{
		ID:         uuid.New(),
		UserID:     1,
		Title:      "Past reminder",
		StartAt:    now,
		ReminderAt: &past,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	// Без напоминания
	eventNoReminder := &domain.Event{
		ID:         uuid.New(),
		UserID:     1,
		Title:      "No reminder",
		StartAt:    now.Add(time.Hour),
		ReminderAt: nil,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	for _, e := range []*domain.Event{eventFuture, eventPast, eventNoReminder} {
		err := testStore.Create(ctx, e)
		require.NoError(t, err)
	}

	pending, err := testStore.GetPendingReminders(ctx, now)
	assert.NoError(t, err)
	assert.Len(t, pending, 1)
	assert.Equal(t, eventFuture.ID, pending[0].ID)
}

// Вспомогательная функция для создания указателя на time.Time
func ptrTime(t time.Time) *time.Time {
	return &t
}
