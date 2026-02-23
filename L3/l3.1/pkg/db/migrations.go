package db

import (
	"context"
	"database/sql"
	"fmt"
)

// schema схема таблицы БД
const schemaNotifications = `CREATE TABLE notifications (
                                 id SERIAL PRIMARY KEY,
                                uid UUID NOT NULL,
                            user_id INT NOT NULL,
                            channel TEXT[] NOT NULL,
                            content TEXT NOT NULL,
                             status VARCHAR(20) NOT NULL DEFAULT 'scheduled',
                           send_for TIMESTAMPTZ NOT NULL,
                            send_at TIMESTAMPTZ,
                        retry_count INT NOT NULL DEFAULT 0,
                         last_error TEXT NOT NULL DEFAULT '',
                         created_at TIMESTAMPTZ NOT NULL DEFAULT NOW());

	                         CREATE UNIQUE INDEX idx_notifications_uid ON notifications(uid);
	                         CREATE INDEX idx_notifications_status_scheduled ON notifications(status, send_for) WHERE status = 'scheduled';`

// Migration создаёт таблицу notifications, если она не существует.
func (c *ClientPostgres) Migration(ctx context.Context) error {

	// проверяем, существует ли таблица (запрос к information_schema)
	var tableName string
	err := c.QueryRowContext(ctx, `SELECT table_name
		                             FROM information_schema.tables
		                            WHERE table_schema = 'public' AND table_name = 'notifications'
									`).Scan(&tableName)

	if err == nil {
		// таблица уже существует
		return nil
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("ошибка проверки существования таблицы: %w", err)
	}

	// создаём таблицу
	_, err = c.ExecContext(ctx, schemaNotifications)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы: %w", err)
	}

	return nil
}
