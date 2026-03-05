package db

import (
	"context"
	"fmt"
)

const (
	linksSchema = `CREATE TABLE IF NOT EXISTS links (
			           id SERIAL PRIMARY KEY,
		        short_url VARCHAR(50) UNIQUE NOT NULL,
		     original_url TEXT NOT NULL,
		       created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		        is_custom BOOLEAN NOT NULL DEFAULT FALSE,
		     clicks_count INT NOT NULL DEFAULT 0);
			 
			 CREATE INDEX IF NOT EXISTS idx_links_short_url ON links(short_url);
		     CREATE INDEX IF NOT EXISTS idx_links_created_at ON links(created_at);`

	analyticsSchema = `CREATE TABLE IF NOT EXISTS analytics (
			               id SERIAL PRIMARY KEY,
			          link_id INT NOT NULL REFERENCES links(id) ON DELETE CASCADE,
			      accessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			       user_agent TEXT,
			       ip_address INET,
			          referer TEXT);
			
				 CREATE INDEX IF NOT EXISTS idx_analytics_link_id_accessed_at ON analytics(link_id, accessed_at);
		         CREATE INDEX IF NOT EXISTS idx_analytics_accessed_at ON analytics(accessed_at);`
)

// Migration создаёт таблицы links и analytics, если они ещё не существуют, добавляет индексы
func (d *DataBase) Migration(ctx context.Context) error {

	// создаём таблицу links с индексами
	query := linksSchema
	_, err := d.Pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы links: %w", err)
	}

	// создаём таблицу analytics с индексами
	query = analyticsSchema
	_, err = d.Pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы analytics: %w", err)
	}

	return nil
}
