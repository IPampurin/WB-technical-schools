package db

import (
	"context"
	"fmt"
)

const (
	eventsSchema = `CREATE TABLE IF NOT EXISTS events (
			            id SERIAL PRIMARY KEY,
			          name TEXT NOT NULL,
				date_event TIMESTAMPTZ NOT NULL,
	   booking_ttl_minutes INT NOT NULL,
	           total_seats INT NOT NULL,
	            free_seats INT NOT NULL,
	         booking_price INT NOT NULL);

	                CREATE INDEX IF NOT EXISTS idx_events_date_event ON events(date_event DESC);`

	usersSchema = `CREATE TABLE IF NOT EXISTS users (
			           id SERIAL PRIMARY KEY,
                     name TEXT,
					email TEXT NOT NULL UNIQUE);
		
	               CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);`

	bookingsSchema = `CREATE TABLE IF NOT EXISTS bookings (
			              id SERIAL PRIMARY KEY,
					event_id INT REFERENCES events(id),
					 user_id INT REFERENCES users(id),
					  status VARCHAR(9),
				  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				  expires_at TIMESTAMPTZ NOT NULL,
				confirmed_at TIMESTAMPTZ);
		
	                  CREATE INDEX IF NOT EXISTS idx_bookings_created_at ON bookings(created_at DESC);`
)

// Migration создаёт таблицы events, users и bookings, если они ещё не существуют, добавляет индексы
func (d *DataBase) Migration(ctx context.Context) error {

	// создаём таблицу events с индексами
	query := eventsSchema
	_, err := d.Pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы events: %w", err)
	}

	// создаём таблицу users с индексами
	query = usersSchema
	_, err = d.Pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы users: %w", err)
	}

	// создаём таблицу bookings с индексами
	query = bookingsSchema
	_, err = d.Pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы bookings: %w", err)
	}

	return nil
}
