// подключение к Postgres и конструктор хранилища
package sqldb

import (
	"context"
	"fmt"
	"time"

	"github.com/IPampurin/EventCalendar/internal/configuration"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxConns        = 25
	minConns        = 5
	maxConnIdle     = 2 * time.Minute
	maxConnLifetime = 30 * time.Minute
)

// Store реализует интерфейс service.EventRepository
type Store struct {
	db *pgxpool.Pool
}

// StartDB открывает пул соединений к Postgres через pgxpool
func StartDB(ctx context.Context, cfg *configuration.DBConfig) (*pgxpool.Pool, error) {

	dsn := cfg.DSN()

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("парсинг DSN: %w", err)
	}

	poolConfig.MaxConns = maxConns
	poolConfig.MinConns = minConns
	poolConfig.MaxConnIdleTime = maxConnIdle
	poolConfig.MaxConnLifetime = maxConnLifetime

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("создание пула: %w", err)
	}

	// проверка соединения
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping БД: %w", err)
	}

	return pool, nil
}

// NewStore возвращает реализацию service.EventRepository с пулом
func NewStore(pool *pgxpool.Pool) *Store {

	return &Store{db: pool}
}
