package zSet

import (
	"context"
	"fmt"

	"github.com/IPampurin/EventBooker/pkg/configuration"

	"github.com/wb-go/wbf/logger"
	"github.com/wb-go/wbf/redis"
)

// ClientZSet хранит подключение к БД Redis и ключ ZSET для просрочек
type ClientZSet struct {
	*redis.Client
	key string
}

// InitZSet запускает работу с Redis и горутину для отслеживания просроченных бронирований
func InitZSet(ctx context.Context, cfg *configuration.ConfZSet, log logger.Logger) (*ClientZSet, <-chan int, error) {

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
		return nil, nil, fmt.Errorf("ошибка установки соединения с реализацией ZSet: %v\n", err)
	}

	// проверяем подключение
	if err := client.Ping(ctx); err != nil {
		return nil, nil, fmt.Errorf("ошибка подключения к реализации ZSet: %v\n", err)
	}

	if cfg.OverdueKey == "" {
		return nil, nil, fmt.Errorf("ошибка инициализации ZSet: задайте ZSET_OVERDUE_KEY\n")
	}

	log.Info("ZSet подключен.")

	// оборачиваем клиент в структуру с методами ZSET
	clientRedis := &ClientZSet{
		Client: client,
		key:    cfg.OverdueKey,
	}

	// канал для передачи номеров просроченных броней брокеру
	ch := make(chan int, 100)

	// запускаем горутину-наблюдатель
	go watchOverdueLoop(ctx, clientRedis, cfg, ch, log)

	return clientRedis, ch, nil
}
