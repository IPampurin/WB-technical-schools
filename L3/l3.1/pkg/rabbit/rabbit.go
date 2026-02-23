package rabbit

import (
	"context"
	"fmt"
	"time"

	"github.com/IPampurin/DelayedNotifier/pkg/configuration"
	"github.com/wb-go/wbf/logger"
	"github.com/wb-go/wbf/rabbitmq"
	"github.com/wb-go/wbf/retry"
)

// ClientRabbit хранит ссылку на оригинальный клиент RabbitMQ и имя очереди
type ClientRabbit struct {
	Client *rabbitmq.RabbitClient
	queue  string
}

// глобальный экземпляр клиента (синглтон)
var defaultClient *ClientRabbit

// InitRabbit инициализирует подключение к RabbitMQ, декларирует очередь и сохраняет глобальный клиент
func InitRabbit(cfg *configuration.ConfRabbitMQ, consumerCfg *configuration.ConfConsumer, log logger.Logger) error {

	// формируем URL для подключения
	amqpURL := fmt.Sprintf("amqp://%s:%s@%s:%d%s", cfg.User, cfg.Password, cfg.HostName, cfg.Port, cfg.VHost) // "amqp://guest:guest@localhost:5672/"

	// создаём стратегии повторных попыток
	reconnectStrategy := retry.Strategy{
		Attempts: consumerCfg.RetryCount,
		Delay:    consumerCfg.RetryDelay,
		Backoff:  float64(consumerCfg.Backoff),
	}
	// для публикации и потребления можно использовать ту же стратегию
	producingStrategy := reconnectStrategy
	consumingStrategy := reconnectStrategy

	// конфигурация клиента RabbitMQ
	clientCfg := rabbitmq.ClientConfig{
		URL:            amqpURL,
		ConnectionName: "delayed-notifier",
		ConnectTimeout: 10 * time.Second,
		Heartbeat:      30 * time.Second,
		ReconnectStrat: reconnectStrategy, // стратегия переподключения при обрыве
		ProducingStrat: producingStrategy, // стратегия повторов при публикации
		ConsumingStrat: consumingStrategy, // стратегия повторов при обработке (для консумера)
	}

	var client *rabbitmq.RabbitClient
	err := retry.Do(func() error {
		var innerErr error
		client, innerErr = rabbitmq.NewClient(clientCfg)
		return innerErr
	}, reconnectStrategy)
	if err != nil {
		return fmt.Errorf("ошибка создания клиента RabbitMQ после %d попыток: %w", reconnectStrategy.Attempts, err)
	}

	ch, err := client.GetChannel()
	if err != nil {
		_ = client.Close()
		return fmt.Errorf("ошибка получения канала: %w", err)
	}
	defer ch.Close()

	_, err = ch.QueueDeclare(
		cfg.Queue, // имя очереди
		true,      // durable (сохранять при перезапуске)
		false,     // autoDelete (не удалять, когда отключатся все потребители)
		false,     // exclusive (не эксклюзивная)
		false,     // noWait
		nil,       // аргументы
	)
	if err != nil {
		_ = client.Close()
		return fmt.Errorf("ошибка объявления очереди: %w", err)
	}

	defaultClient = &ClientRabbit{
		Client: client,
		queue:  cfg.Queue,
	}

	log.Info("RabbitMQ запущен", "queue", cfg.Queue)

	return nil
}

// GetClient возвращает глобальный экземпляр клиента Rabbit
func GetClient() *ClientRabbit {
	return defaultClient
}

// Publish публикует сообщение в заданную очередь (Publisher с exchange = "")
func (c *ClientRabbit) Publish(ctx context.Context, body []byte) error {

	publisher := rabbitmq.NewPublisher(c.Client, "", "application/json")
	// публикуем напрямую в очередь, routingKey = имя очереди
	return publisher.Publish(ctx, body, c.queue)
}

// GetQueueName возвращает имя очереди (будет для консумера)
func (c *ClientRabbit) GetQueueName() string {
	return c.queue
}
