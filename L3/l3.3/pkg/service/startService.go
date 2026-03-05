package service

import (
	"context"

	"github.com/IPampurin/UrlShortener/pkg/cache"
	"github.com/IPampurin/UrlShortener/pkg/db"
)

type Service struct {
	link      db.LinkMethods
	analytics db.AnalyticsMethods
	cache     cache.CacheMethods
}

func InitService(ctx context.Context, storage *db.DataBase, cache *cache.Cache) *Service {

	svc := &Service{
		link:      storage, // *db.DataBase реализует LinkMethods
		analytics: storage, // *db.DataBase реализует AnalyticsMethods
		cache:     cache,   // *cache.Cache реализует CacheMethods
	}

	return svc
}
