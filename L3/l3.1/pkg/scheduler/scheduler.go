package scheduler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/IPampurin/DelayedNotifier/pkg/cache"
	"github.com/IPampurin/DelayedNotifier/pkg/configuration"
	"github.com/IPampurin/DelayedNotifier/pkg/db"
	"github.com/IPampurin/DelayedNotifier/pkg/rabbit"
	"github.com/wb-go/wbf/logger"
)

// InitScheduler запускает планировщик с заданным интервалом ConfScheduler.Interval,
// собирает уведомления из БД, готовые к отправке, и публикует их в rabbitmq
func InitScheduler(ctx context.Context, cfg *configuration.ConfScheduler, log logger.Logger) error {

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	log.Info("планировщик запущен", "интервал", cfg.Interval)

	for {
		select {
		case <-ctx.Done():
			log.Info("планировщик завершает работу")
			return nil
		case <-ticker.C:
			// получаем список уведомлений, которые уже пора отправлять
			notifications, err := db.GetClientDB().GetScheduledNotifications(ctx, cfg.Interval)
			if err != nil {
				log.Error("ошибка получения уведомлений из БД", "error", err)
				continue
			}
			// если подходящих уведомлений нет
			if len(notifications) == 0 {
				continue
			}
			// обрабатываем каждое уведомление
			for _, n := range notifications {
				if err := publishToRabbit(ctx, n, log); err != nil {
					log.Error("ошибка публикации в брокер", "uid", n.UID, "error", err)
				}
			}
		}
	}
}

// publishToRabbit публикует уведомление в очередь ConfRabbitMQ.Queue брокера:
// - при успехе меняет статус на publishing и удаляет ключ из кэша,
// - при ошибке публикации меняет статус на failed и также удаляет ключ из кэша
func publishToRabbit(ctx context.Context, n *db.Notification, log logger.Logger) error {

	// сериализуем уведомление в json
	body, err := json.Marshal(n)
	if err != nil {
		return err // ошибка маршалинга, дальше не идём
	}

	// публикуем сообщение (внутри уже есть ретраи)
	err = rabbit.GetClient().Publish(ctx, body)
	if err != nil {
		// ошибка публикации: обновляем статус на failed и логируем
		if updErr := db.GetClientDB().UpdateNotificationStatus(ctx, n.UID, db.StatusFailed, nil, n.RetryCount, "publish failed: "+err.Error()); updErr != nil {
			log.Error("не удалось обновить статус на failed", "uid", n.UID, "error", updErr)
		}
		// инвалидируем кэш, чтобы при следующем get подтянулись актуальные данные
		if cacheClient := cache.GetClientRedis(); cacheClient != nil {
			if delErr := cacheClient.Del(ctx, n.UID.String()); delErr != nil {
				log.Error("не удалось удалить ключ из кэша", "uid", n.UID, "error", delErr)
			}
		}
		return err // возвращаем исходную ошибку публикации
	}

	// публикация успешна: меняем статус на publishing
	if updErr := db.GetClientDB().UpdateNotificationStatus(ctx, n.UID, db.StatusPublishing, nil, n.RetryCount, ""); updErr != nil {
		log.Error("не удалось обновить статус на publishing", "uid", n.UID, "error", updErr)
	}
	// удаляем из кэша, чтобы следующий get получил свежий статус из бд
	if cacheClient := cache.GetClientRedis(); cacheClient != nil {
		if delErr := cacheClient.Del(ctx, n.UID.String()); delErr != nil {
			log.Error("не удалось удалить ключ из кэша", "uid", n.UID, "error", delErr)
		}
	}

	log.Info("уведомление отправлено в очередь", "uid", n.UID)

	return nil
}
