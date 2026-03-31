package sqldb

import (
	"context"
	"fmt"
	"time"
)

// ArchiveOlderThan переносит прошедшие (StartAt, при EndAt == nil) и закончившиеся (EndAt) события в архив
// и удаляет их из активной таблицы, mark - время начала процедуры архивации (записывается в archived_at),
// использует один атомарный запрос, возвращает количество заархивированных событий
func (s *Store) ArchiveOlderThan(ctx context.Context, mark time.Time) (int, error) {

	query := `WITH deleted AS (
			                   DELETE
							     FROM events
			                    WHERE (end_at IS NOT NULL AND end_at < $1)
			                          OR (end_at IS NULL AND start_at < $1)
			                   RETURNING *
		                       )
		      INSERT INTO archive_events (id, 
			                              user_id,
										  title,
										  description,
										  start_at,
										  end_at,
										  reminder_at,
										  created_at,
										  updated_at,
										  archived_at)
		      SELECT id,
			         user_id,
					 title,
					 description,
					 start_at,
					 end_at,
					 reminder_at,
					 created_at,
					 updated_at,
					 $1
		        FROM deleted`

	cmdTag, err := s.db.Exec(ctx, query, mark)
	if err != nil {
		return 0, fmt.Errorf("ошибка архивации: %w", err)
	}

	return int(cmdTag.RowsAffected()), nil
}
