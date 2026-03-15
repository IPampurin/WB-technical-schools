package db

import (
	"context"
	"fmt"
)

const (
	itemsSchema = `CREATE TABLE IF NOT EXISTS items (
			            id SERIAL PRIMARY KEY,
			      category TEXT NOT NULL,
				    amount DECIMAL,
				      date TIMESTAMPTZ NOT NULL);

					CREATE INDEX IF NOT EXISTS idx_items_category ON items(category);  
	                CREATE INDEX IF NOT EXISTS idx_items_amount ON items(amount);
 				    CREATE INDEX IF NOT EXISTS idx_items_date ON items(date);`
)

// Migration создаёт таблицу items, если она ещё не существуют, добавляет индексы
func (d *DataBase) Migration(ctx context.Context) error {

	// создаём таблицу items с индексами
	query := itemsSchema
	_, err := d.Pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы items: %w", err)
	}

	return nil
}
