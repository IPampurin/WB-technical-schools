package cache

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/IPampurin/DelayedNotifier/pkg/configuration"
	"github.com/IPampurin/DelayedNotifier/pkg/db"

	"github.com/wb-go/wbf/redis"
	"github.com/wb-go/wbf/retry"
)

// ClientRedis хранит подключение к БД Redis
// делаем его публичным, чтобы другие пакеты могли использовать методы
type ClientRedis struct {
	*redis.Client
}

// глобальный экземпляр клиента (синглтон)
var defaultClient *ClientRedis

// InitRedis запускает работу с Redis
func InitRedis(cfg *configuration.ConfCache) error {

	// определяем конфигурацию подключения к Redis
	options := redis.Options{
		Address:   fmt.Sprintf("%s:%d", cfg.HostName, cfg.Port),
		Password:  cfg.Password,
		MaxMemory: "100mb",
		Policy:    "allkeys-lru",
	}

	// пробуем подключиться
	client, err := redis.Connect(options)
	if err != nil {
		return fmt.Errorf("ошибка установки соединения с Redis: %v\n", err)
	}

	// проверяем подключение
	if err := client.Ping(context.Background()); err != nil {
		return fmt.Errorf("ошибка подключения к Redis: %v\n", err)
	}

	// сохраняем в глобальную переменную
	defaultClient = &ClientRedis{client}

	// загружаем начальные данные
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = loadDataToCache(ctx, cfg)
	if err != nil {
		log.Printf("ошибка загрузки первичных данных в кэш: %v", err)
		return err
	}

	log.Println("Кэш прогрет и работает.")

	return nil
}

// GetClientRedis возвращает глобальный экземпляр клиента Redis
func GetClientRedis() *ClientRedis {

	return defaultClient
}

// loadDataToCache загружает данные за последнее время в кэш при старте
func loadDataToCache(ctx context.Context, cfg *configuration.ConfCache) error {

	// получаем заказы до установленного порога
	notifications, err := db.GetClientDB().GetNotificationsLastPeriod(ctx, cfg.Warming)
	if err != nil {
		return fmt.Errorf("ошибка получения уведомлений из БД при прогреве кэша: %v", err)
	}
	// проверяем были ли уведомления в БД
	if len(notifications) == 0 {
		return nil
	}

	// определяем стратегию ретраев
	strategy := retry.Strategy{Attempts: 3, Delay: 100 * time.Millisecond, Backoff: 2}

	// сохраняем данные в redis
	for i := range notifications {
		key := notifications[i].UID.String()
		err := GetClientRedis().SetWithExpirationAndRetry(ctx, strategy, key, notifications[i], cfg.TTL)
		if err != nil {
			log.Printf("ошибка добавления уведомления %s при прогреве кэша", key)
			continue
		}
	}

	return nil
}
