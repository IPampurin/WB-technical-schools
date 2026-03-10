package db

import (
	"context"
	"fmt"
)

const (
	imagesSchema = `CREATE TABLE IF NOT EXISTS images (
			            id UUID PRIMARY KEY,
			   original_id UUID REFERENCES images(id) ON DELETE CASCADE,
			          name TEXT NOT NULL,
			          type TEXT NOT NULL,
			  content_type TEXT NOT NULL,
			          size BIGINT NOT NULL,
			         width INT,
			        height INT,
			        status TEXT NOT NULL,
			 error_message TEXT,
			  storage_path TEXT NOT NULL,
			    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW());
		
	                CREATE INDEX IF NOT EXISTS idx_images_original_id ON images(original_id);
	                CREATE INDEX IF NOT EXISTS idx_images_created_at ON images(created_at DESC);`

	outboxSchema = `CREATE TABLE IF NOT EXISTS outbox (
			            id UUID PRIMARY KEY,
			         topic TEXT NOT NULL,
			           key TEXT NOT NULL,
			       payload BYTEA NOT NULL,
			    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW());
		
	                CREATE INDEX IF NOT EXISTS idx_outbox_created_at ON outbox(created_at);`
)

// Migration создаёт таблицы images и outbox, если они ещё не существуют, добавляет индексы
func (d *DataBase) Migration(ctx context.Context) error {

	// создаём таблицу images с индексами
	query := imagesSchema
	_, err := d.Pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы images: %w", err)
	}

	// создаём таблицу outbox с индексами
	query = outboxSchema
	_, err = d.Pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы outbox: %w", err)
	}

	return nil
}
