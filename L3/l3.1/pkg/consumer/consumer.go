package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/IPampurin/DelayedNotifier/pkg/cache"
	"github.com/IPampurin/DelayedNotifier/pkg/configuration"
	"github.com/IPampurin/DelayedNotifier/pkg/db"
	"github.com/IPampurin/DelayedNotifier/pkg/rabbit"
	"github.com/rabbitmq/amqp091-go"
	"github.com/wb-go/wbf/logger"
	"github.com/wb-go/wbf/rabbitmq"
	"github.com/wb-go/wbf/retry"
)

// InitConsumer запускает консумер очереди RabbitMQ
func InitConsumer(ctx context.Context, cfg *configuration.ConfConsumer, log logger.Logger) error {

	rabbitClient := rabbit.GetClient()
	if rabbitClient == nil {
		return fmt.Errorf("rabbitmq клиент не инициализирован")
	}

	// конфигурация потребителя
	consumerCfg := rabbitmq.ConsumerConfig{
		Queue:         rabbitClient.GetQueueName(),
		ConsumerTag:   "delayed-notifier",
		AutoAck:       false, // ручное подтверждение
		Ask:           rabbitmq.AskConfig{Multiple: false},
		Nack:          rabbitmq.NackConfig{Multiple: false, Requeue: false}, // по умолчанию не возвращаем в очередь
		Args:          nil,
		Workers:       1,
		PrefetchCount: 1,
	}

	// обработчик сообщений
	handler := func(ctx context.Context, delivery amqp091.Delivery) error {
		return handleMessage(ctx, cfg, delivery, log)
	}

	// создаём консумера, используя Client
	consumer := rabbitmq.NewConsumer(rabbitClient.Client, consumerCfg, handler)

	log.Info("запуск консумера", "queue", consumerCfg.Queue, "workers", consumerCfg.Workers)

	// запускаем консумер (блокируется до отмены контекста или ошибки)
	// в случае ошибки возвращаем её, и из main отменяется общий контекст
	if err := consumer.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	return nil
}

// handleMessage обрабатывает одно сообщение из очереди
func handleMessage(ctx context.Context, cfg *configuration.ConfConsumer, delivery amqp091.Delivery, log logger.Logger) error {

	// 1. Парсим уведомление
	var n db.Notification
	if err := json.Unmarshal(delivery.Body, &n); err != nil {
		log.Error("ошибка разбора сообщения", "error", err)
		// битое сообщение — удаляем из очереди
		return delivery.Nack(false, false)
	}

	log.Info("получено уведомление из очереди", "uid", n.UID)

	// 2. Получаем актуальное состояние из БД (на случай, если уведомление уже отменено)
	current, err := db.GetClientDB().GetNotification(ctx, n.UID)
	if err != nil {
		log.Error("не удалось получить уведомление из БД", "uid", n.UID, "error", err)
		// если БД недоступна, вернём сообщение в очередь, чтобы попробовать позже
		return delivery.Nack(false, true)
	}

	// 3. Если статус уже не publishing (например, отменено) — просто подтверждаем
	if current.Status != db.StatusPublishing {
		log.Info("уведомление уже не в статусе publishing, пропускаем", "uid", n.UID, "status", current.Status)
		return delivery.Ack(false)
	}

	// 4. Отправляем по каналам
	success, lastErr := sendByChannels(ctx, cfg, current, log)

	// 5. Обновляем статус в БД и кэш
	now := time.Now()
	if success {
		// хотя бы один канал сработал
		if err := db.GetClientDB().UpdateNotificationStatus(ctx, n.UID, db.StatusSent, &now, current.RetryCount, ""); err != nil {
			log.Error("не удалось обновить статус на sent", "uid", n.UID, "error", err)
		}
		log.Info("уведомление успешно отправлено", "uid", n.UID)
	} else {
		// все каналы не сработали
		if err := db.GetClientDB().UpdateNotificationStatus(ctx, n.UID, db.StatusFailed, nil, current.RetryCount, lastErr); err != nil {
			log.Error("не удалось обновить статус на failed", "uid", n.UID, "error", err)
		}
		log.Error("уведомление не отправлено", "uid", n.UID, "lastError", lastErr)
	}

	// инвалидируем кэш (или можно обновить, но удаление проще)
	if cacheClient := cache.GetClientRedis(); cacheClient != nil {
		if delErr := cacheClient.Del(ctx, n.UID.String()); delErr != nil {
			log.Error("не удалось удалить ключ из кэша", "uid", n.UID, "error", delErr)
		}
	}

	// подтверждаем обработку сообщения (оно удаляется из очереди)
	return delivery.Ack(false)
}

// sendByChannels пытается отправить уведомление по всем указанным каналам,
// используя повторные попытки с экспоненциальной задержкой,
// возвращает true, если хотя бы одна отправка успешна, иначе false + текст последней ошибки
func sendByChannels(ctx context.Context, cfg *configuration.ConfConsumer, n *db.Notification, log logger.Logger) (success bool, lastErr string) {

	// если каналы не указаны, считаем ошибкой (хоть фронт и проверяет)
	if len(n.Channel) == 0 {
		return false, "каналы отправки не указаны"
	}

	// стратегия повторных попыток из глобальных переменных
	strategy := retry.Strategy{
		Attempts: cfg.RetryCount,
		Delay:    cfg.RetryDelay,
		Backoff:  float64(cfg.Backoff),
	}

	// переменная для хранения результата: был ли хоть один успех
	anySuccess := false

	// пытаемся отправить по каждому каналу
	for _, ch := range n.Channel {
		// для каждого канала своя функция отправки с ретраями
		err := retry.DoContext(ctx, strategy, func() error {
			var sendErr error
			switch ch {
			case "email":
				sendErr = sendEmail(n)
			case "telegram":
				sendErr = sendTelegram(n)
			default:
				sendErr = fmt.Errorf("невалидный канал отправки: %s", ch)
			}
			if sendErr != nil {
				log.Warn("ошибка отправки, попытка повтора",
					"channel", ch,
					"uid", n.UID,
					"error", sendErr)
			}
			return sendErr
		})

		if err == nil {
			// отправка по данному каналу успешна
			anySuccess = true
			log.Info("уведомление успешно отправлено по каналу",
				"channel", ch,
				"uid", n.UID)
			// не прерываем цикл — отправляем по всем каналам
		} else {
			// после всех ретраев канал не сработал
			log.Error("не удалось отправить по каналу после всех попыток",
				"channel", ch,
				"uid", n.UID,
				"error", err)
			lastErr = err.Error() // сохраняем последнюю ошибку
		}
	}

	if anySuccess {
		return true, ""
	}
	// если ни один канал не сработал, возвращаем false и последнюю ошибку
	if lastErr == "" {
		lastErr = "отправка не удалась"
	}
	return false, lastErr
}

// заглушки для реальной отправки
// sendEmail отправляет уведомление по почте
func sendEmail(_ *db.Notification) error {
	// TODO: реализовать отправку email
	// для тестирования возвращаем ошибку
	return fmt.Errorf("email sending not implemented")
}

// sendTelegram отправляет уведомление в телеграм
func sendTelegram(_ *db.Notification) error {
	// TODO: реализовать отправку telegram
	return fmt.Errorf("telegram sending not implemented")
}
