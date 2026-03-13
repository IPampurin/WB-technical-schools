package zSet

import (
	"context"
	"strconv"
	"time"

	"github.com/IPampurin/EventBooker/pkg/configuration"
	"github.com/wb-go/wbf/logger"
	"github.com/wb-go/wbf/redis"
	"github.com/wb-go/wbf/retry"
)

// watchOverdueLoop периодически проверяет ZSET и отправляет найденные ID в канал
func watchOverdueLoop(ctx context.Context, client *ClientZSet, cfg *configuration.ConfZSet, ch chan<- int, log logger.Logger) {

	defer close(ch)      // закрываем канал при выходе
	defer client.Close() // закрываем соединение с ZSET

	ticker := time.NewTicker(cfg.CheckInterval)
	defer ticker.Stop()

	// стратегия повторов для чтения из ZSET
	readRetry := retry.Strategy{
		Attempts: cfg.ReadRetryAttempts,
		Delay:    cfg.ReadRetryDelay,
		Backoff:  cfg.ReadRetryBackoff,
	}

	for {
		select {
		case <-ctx.Done():
			log.Info("горутина наблюдателя ZSET остановлена по контексту")
			return

		case <-ticker.C:
			now := time.Now().Unix()

			// читаем из ZSET с повторными попытками
			var members []string
			err := retry.DoContext(ctx, readRetry, func() error {
				var e error
				members, e = client.ZRangeByScore(ctx, 0, now)
				if e != nil && e != redis.NoMatches {
					return e // любая ошибка, кроме "нет элементов", запустит повтор
				}
				return nil // успех или redis.NoMatches (нет элементов)
			})

			if err != nil {
				log.Error("ошибка чтения ZSET после всех попыток", "error", err)
				continue
			}

			// обрабатываем каждый найденный элемент
			for _, member := range members {

				id, err := strconv.Atoi(member)
				if err != nil {
					log.Error("неверный формат ID в ZSET", "value", member)
					continue
				}

				sendTimeout := 1 * time.Second // таймаут отправки в канал

				// пытаемся отправить ID в канал брокера с таймаутом
				select {

				case <-ctx.Done(): // контекст отменен - уходим
					return

				case ch <- id: // успешно отправили - удаляем элемент из ZSET
					if err := client.ZRem(ctx, member); err != nil {
						log.Error("не удалось удалить элемент из ZSET", "id", id, "error", err)
					} else {
						log.Info("просроченная бронь отправлена и удалена из ZSET", "id", id)
					}

				case <-time.After(sendTimeout): // таймаут отправки
					log.Error("таймаут отправки ID в канал брокера", "id", id)
					// не удаляем элемент - он будет обработан при следующем цикле
				}
			}
		}
	}
}
