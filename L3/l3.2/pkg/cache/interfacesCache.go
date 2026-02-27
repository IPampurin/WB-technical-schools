package cache

import (
	"context"

	"github.com/IPampurin/UrlShortener/pkg/db"
)

type CacheMethods interface {
	// GetLink возвращает ссылку из кэша по её короткому URL
	GetLink(ctx context.Context, shortURL string) (*db.Link, error)

	// SetLink сохраняет ссылку в кэш с предустановленным TTL
	SetLink(ctx context.Context, shortURL string, link *db.Link) error

	// DeleteLink удаляет ссылку из кэша
	DeleteLink(ctx context.Context, shortURL string) error

	// LoadDataToCache выполняет прогрев кэша, сохраняя переданный список ссылок
	LoadDataToCache(ctx context.Context, lastLinks []*db.Link) error
}
