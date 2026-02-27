package db

import (
	"context"
	"time"
)

// методы по таблице Link
type LinkMethods interface {
	// CreateLink создаёт новую запись в таблице links
	CreateLink(ctx context.Context, originalURL, shortURL string, isCustom bool) (*Link, error)

	// GetLinkByShortURL возвращает ссылку по её короткому идентификатору
	GetLinkByShortURL(ctx context.Context, shortURL string) (*Link, error)

	// GetLinkByOriginalURL возвращает все ссылки, соответствующие заданному оригинальному URL
	GetLinkByOriginalURL(ctx context.Context, originalURL string) ([]*Link, error)

	// IncrementClicks увеличивает счётчик переходов по ссылке на единицу
	IncrementClicks(ctx context.Context, linkID int64) error

	// GetLinks возвращает последние 20 созданных ссылок
	GetLinks(ctx context.Context) ([]*Link, error)

	// GetLinksOfPeriod возвращает ссылки, созданные за указанный период времени
	GetLinksOfPeriod(ctx context.Context, period time.Duration) ([]*Link, error)

	// SearchByOriginalURL ищет ссылки, OriginalURL которых содержит подстроку query
	SearchByOriginalURL(ctx context.Context, search string) ([]*Link, error)

	// SearchByShortURL ищет ссылки, ShortURL которых содержит подстроку query
	SearchByShortURL(ctx context.Context, search string) ([]*Link, error)
}

// методы по таблице Analytics
type AnalyticsMethods interface {
	// SaveAnalytics сохраняет информацию о переходе по ссылке
	SaveAnalytics(ctx context.Context, linkID int, accessedAt time.Time, userAgent, ipAddress, referer string) error

	// GetAnalyticsByLinkID возвращает все записи о переходах для конкретной ссылки
	GetAnalyticsByLinkID(ctx context.Context, linkID int) ([]*Analytics, error)

	// CountClicksByDay возвращает количество переходов по ссылке, сгруппированных по дням в заданном диапазоне
	CountClicksByDay(ctx context.Context, linkID int, from, to time.Time) (map[string]int, error)

	// CountClicksByMonth возвращает количество переходов по ссылке, сгруппированных по месяцам в заданном диапазоне
	CountClicksByMonth(ctx context.Context, linkID int, from, to time.Time) (map[string]int, error)

	// CountClicksByUserAgent возвращает количество переходов по ссылке, сгруппированных по User-Agent
	CountClicksByUserAgent(ctx context.Context, linkID int) (map[string]int, error)
}
