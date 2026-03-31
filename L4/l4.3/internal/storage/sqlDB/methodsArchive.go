// реализация service.EventRepository для Postgres
package sqldb

import (
	"context"
	"fmt"

	"github.com/IPampurin/EventCalendar/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ArchiveEventCreate вставляет событие в архивную таблицу
func (s *Store) ArchiveEventCreate(ctx context.Context, e *domain.ArchiveEvent) error {

	dbArch := mapArchiveEventToDB(e)
	query := `INSERT INTO archive_events (id, 
	                                      user_id, title, 
										  description, 
										  start_at, 
										  end_at, 
										  reminder_at, 
										  created_at, 
										  updated_at, 
										  archived_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := s.db.Exec(ctx, query,
		dbArch.ID,
		dbArch.UserID,
		dbArch.Title,
		dbArch.Description,
		dbArch.StartAt,
		dbArch.EndAt,
		dbArch.ReminderAt,
		dbArch.CreatedAt,
		dbArch.UpdatedAt,
		dbArch.ArchivedAt,
	)
	if err != nil {
		return fmt.Errorf("ошибка вставки в архив: %w", err)
	}

	return nil
}

// GetArchiveByID возвращает архивное событие по ID
func (s *Store) GetArchiveByID(ctx context.Context, userID int64, eventID uuid.UUID) (*domain.ArchiveEvent, error) {

	query := `SELECT id, 
	                 user_id, 
					 title, 
					 description, 
					 start_at, 
					 end_at, 
					 reminder_at, 
					 created_at, 
					 updated_at, 
					 archived_at
			    FROM archive_events
			   WHERE id = $1 
			         AND user_id = $2`

	var dbArch ArchiveEvent
	err := s.db.QueryRow(ctx, query, eventID, userID).Scan(
		&dbArch.ID,
		&dbArch.UserID,
		&dbArch.Title,
		&dbArch.Description,
		&dbArch.StartAt,
		&dbArch.EndAt,
		&dbArch.ReminderAt,
		&dbArch.CreatedAt,
		&dbArch.UpdatedAt,
		&dbArch.ArchivedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("ошибка получения архивного события: %w", err)
	}

	return mapDBToArchiveEvent(dbArch), nil
}

// GetAllArchive возвращает все архивные события пользователя (пагинация: limit, offset)
func (s *Store) GetAllArchive(ctx context.Context, userID int64, limit, offset int) ([]*domain.ArchiveEvent, error) {

	query := `SELECT id, 
	                 user_id, 
					 title, 
					 description, 
					 start_at, 
					 end_at, 
					 reminder_at, 
					 created_at, 
					 updated_at, 
					 archived_at
			    FROM archive_events 
			   WHERE user_id = $1 
			   ORDER BY archived_at DESC 
			   LIMIT $2 
			  OFFSET $3`

	rows, err := s.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("ошибка выборки архива: %w", err)
	}
	defer rows.Close()

	events := make([]*domain.ArchiveEvent, 0)
	for rows.Next() {
		var dbArch ArchiveEvent
		err := rows.Scan(
			&dbArch.ID,
			&dbArch.UserID,
			&dbArch.Title,
			&dbArch.Description,
			&dbArch.StartAt,
			&dbArch.EndAt,
			&dbArch.ReminderAt,
			&dbArch.CreatedAt,
			&dbArch.UpdatedAt,
			&dbArch.ArchivedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования архивной строки: %w", err)
		}
		events = append(events, mapDBToArchiveEvent(dbArch))
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации: %w", err)
	}

	return events, nil
}
