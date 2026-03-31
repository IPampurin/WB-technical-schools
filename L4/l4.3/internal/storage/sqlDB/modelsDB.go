// модели, с которыми работает БД
package sqldb

import (
	"time"

	"github.com/google/uuid"
)

// Event - запись в таблице events
type Event struct {
	ID          uuid.UUID  `db:"id"`          // uid события
	UserID      int64      `db:"user_id"`     // id пользователя, к которому относится событие
	Title       string     `db:"title"`       // название события
	Description string     `db:"description"` // описание события
	StartAt     time.Time  `db:"start_at"`    // начало события
	EndAt       *time.Time `db:"end_at"`      // окончание события, nil - точечное событие без длительности
	ReminderAt  *time.Time `db:"reminder_at"` // момент когда надо отправить напоминание, nil - напоминание не нужно
	CreatedAt   time.Time  `db:"created_at"`  // время создания записи
	UpdatedAt   time.Time  `db:"updated_at"`  // время последнего обновления
}

// ArchiveEvent - запись в таблице archiv_events
type ArchiveEvent struct {
	ID          uuid.UUID  `db:"id"`          // uid события
	UserID      int64      `db:"user_id"`     // id пользователя, к которому относится событие
	Title       string     `db:"title"`       // название события
	Description string     `db:"description"` // описание события
	StartAt     time.Time  `db:"start_at"`    // начало события
	EndAt       *time.Time `db:"end_at"`      // окончание события, nil - точечное событие без длительности
	ReminderAt  *time.Time `db:"reminder_at"` // момент когда надо отправить напоминание, nil - напоминание не нужно
	CreatedAt   time.Time  `db:"created_at"`  // время создания записи
	UpdatedAt   time.Time  `db:"updated_at"`  // время последнего обновления
	ArchivedAt  time.Time  `db:"archived_at"` // время архивации
}
