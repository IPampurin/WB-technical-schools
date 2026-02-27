package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/IPampurin/UrlShortener/pkg/configuration"
	"github.com/IPampurin/UrlShortener/pkg/db"
	"github.com/wb-go/wbf/logger"
	"github.com/wb-go/wbf/redis"
)

/*
в кэше должны хранится пары ShortURL - OriginalURL
также должны быть сочетания OriginalURL - ShortURL // при чём ShortURL в этом случае только сгенерированные! (кастомные ShortURL надо проверять первыми)
*/

// Cache хранит подключение к БД Redis
type Cache struct {
	redis   *redis.Client
	ttl     time.Duration
	warming time.Duration
}

// InitCache запускает работу с Redis
func InitCache(ctx context.Context, storage *db.DataBase, cfgCache *configuration.ConfCache, log logger.Logger) (*Cache, error) {

	// определяем конфигурацию подключения к Redis
	options := redis.Options{
		Address:   fmt.Sprintf("%s:%d", cfgCache.HostName, cfgCache.Port),
		Password:  cfgCache.Password,
		MaxMemory: "100mb",
		Policy:    "allkeys-lru",
	}

	// пробуем подключиться
	clientRedis, err := redis.Connect(options)
	if err != nil {
		return nil, fmt.Errorf("ошибка установки соединения с Redis: %v\n", err)
	}

	// проверяем подключение
	err = clientRedis.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к Redis: %v\n", err)
	}

	// получаем экземпляр
	cache := &Cache{
		redis:   clientRedis,
		ttl:     cfgCache.TTL,
		warming: cfgCache.Warming,
	}

	// прогреваем кэш, если с ним всё норм
	if cache.redis != nil {

		// получаем список крайних записей
		links, err := storage.GetLinksOfPeriod(ctx, cache.warming)
		if err != nil {
			log.Warn("ошибка прогрева кэша", "error", err)
		}

		// грузим записи в кэш
		err = cache.LoadDataToCache(ctx, links)
		if err != nil {
			log.Warn("ошибка прогрева кэша", "error", err)
		}
	}

	log.Info("Кэш работает.")

	return cache, nil
}
