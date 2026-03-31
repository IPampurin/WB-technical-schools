// структуры базовых сущностей
package domain

import (
	"time"

	"github.com/google/uuid"
)

// Event - событие, сущность календаря
type Event struct {
	ID          uuid.UUID  // uid события
	UserID      int64      // id пользователя, к которому относится событие
	Title       string     // название события
	Description string     // описание события
	StartAt     time.Time  // начало события
	EndAt       *time.Time // окончание события, nil - точечное событие без длительности
	ReminderAt  *time.Time // момент когда надо отправить напоминание, nil - напоминание не нужно
	CreatedAt   time.Time  // время создания записи
	UpdatedAt   time.Time  // время последнего обновления
}

// ArchiveEvent - архивное событие
type ArchiveEvent struct {
	ID          uuid.UUID  // uid события
	UserID      int64      // id пользователя, к которому относится событие
	Title       string     // название события
	Description string     // описание события
	StartAt     time.Time  // начало события
	EndAt       *time.Time // окончание события, nil - точечное событие без длительности
	ReminderAt  *time.Time // момент когда надо отправить напоминание, nil - напоминание не нужно
	CreatedAt   time.Time  // время создания записи
	UpdatedAt   time.Time  // время последнего обновления
	ArchivedAt  time.Time  // время архивации
}

// ReminderTask - задача для напоминальщика (канал/планировщик)
type ReminderTask struct {
	EventID  uuid.UUID // uid события, для которого срабатывает напоминание
	UserID   int64     // id пользователя
	RemindAt time.Time // момент отправки напоминания
	Title    string    // краткий текст для уведомления (например, название события)
}
