package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/IPampurin/UrlShortener/pkg/db"
	"github.com/wb-go/wbf/redis"
	"github.com/wb-go/wbf/retry"
)

// LoadDataToCache загружает данные за последнее время в кэш при старте
func (c *Cache) LoadDataToCache(ctx context.Context, lastLinks []*db.Link) error {

	strategy := retry.Strategy{Attempts: 3, Delay: 100 * time.Millisecond, Backoff: 2}

	for _, link := range lastLinks {

		key := link.ShortURL
		data, err := json.Marshal(link)
		if err != nil {
			log.Printf("ошибка маршалинга ссылки %s при прогреве кэша: %v", key, err)
			continue
		}

		err = c.redis.SetWithExpirationAndRetry(ctx, strategy, key, data, c.ttl)
		if err != nil {
			log.Printf("ошибка добавления ссылки %s при прогреве кэша: %v", key, err)
			continue
		}
	}

	return nil
}

// GetLink возвращает ссылку из кэша по короткому URL (или nil, nil)
func (c *Cache) GetLink(ctx context.Context, shortURL string) (*db.Link, error) {

	data, err := c.redis.Get(ctx, shortURL)
	if err != nil {
		if errors.Is(err, redis.NoMatches) {
			return nil, nil
		}
		return nil, err
	}

	var link db.Link
	if err := json.Unmarshal([]byte(data), &link); err != nil {
		return nil, err
	}

	return &link, nil
}

// SetLink сохраняет ссылку в кэш с внутренним TTL
func (c *Cache) SetLink(ctx context.Context, shortURL string, link *db.Link) error {

	data, err := json.Marshal(link)
	if err != nil {
		return err
	}

	return c.redis.SetWithExpiration(ctx, shortURL, data, c.ttl)
}

// DeleteLink удаляет ссылку из кэша
func (c *Cache) DeleteLink(ctx context.Context, shortURL string) error {

	return c.redis.Del(ctx, shortURL)
}
