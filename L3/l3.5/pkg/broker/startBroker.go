package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/IPampurin/EventBooker/pkg/configuration"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/wb-go/wbf/logger"
	"github.com/wb-go/wbf/rabbitmq"
	"github.com/wb-go/wbf/retry"
)

// Broker содержит канал для получения просроченных броней и метод закрытия
type Broker struct {
	messages <-chan int
	close    func() error
}

// Messages возвращает канал, из которого сервисный слой читает ID просроченных броней
func (b *Broker) Messages() <-chan int {

	return b.messages
}

// Close закрывает соединение с RabbitMQ
func (b *Broker) Close() error {

	if b.close != nil {
		return b.close()
	}

	return nil
}

// InitBroker создаёт подключение к RabbitMQ, объявляет exchange/очередь, запускает горутину для публикации
// просрочек из канала zSet и consumer для чтения очереди, возвращает Broker, содержащий канал для сервисного слоя
func InitBroker(ctx context.Context, cfg *configuration.ConfBroker, overdueCh <-chan int, log logger.Logger) (*Broker, error) {

	// 1. Формируем URL для подключения к RabbitMQ
	amqpURL := fmt.Sprintf("amqp://%s:%s@%s:%d%s",
		cfg.User, cfg.Password, cfg.HostName, cfg.Port, cfg.VHost)

	// 2. Настраиваем единую стратегию повторных попыток
	retryStrategy := retry.Strategy{
		Attempts: cfg.RetryAttempts,
		Delay:    cfg.RetryDelay,
		Backoff:  cfg.RetryBackoff,
	}

	// 3. Конфигурация клиента RabbitMQ
	clientCfg := rabbitmq.ClientConfig{
		URL:            amqpURL,
		ConnectionName: cfg.ConnName,
		ConnectTimeout: cfg.ConnTimeout,
		Heartbeat:      cfg.Heartbeat,
		ReconnectStrat: retryStrategy, // стратегия для автоматического переподключения при обрыве
		ProducingStrat: retryStrategy, // стратегия для публикации сообщений
		ConsumingStrat: retryStrategy, // стратегия для потребления сообщений
	}

	// 4. Создаём клиента RabbitMQ
	client, err := rabbitmq.NewClient(clientCfg)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания клиента RabbitMQ: %w", err)
	}

	// 5. Объявляем exchange (точку обмена) с типом "direct" и признаком durable (переживает перезапуск)
	if err = client.DeclareExchange(cfg.Exchange, "direct", true, false, false, nil); err != nil {
		_ = client.Close() // при ошибке закрываем клиент, чтобы не оставить открытое соединение
		return nil, fmt.Errorf("ошибка объявления exchange: %w", err)
	}

	// 6. Объявляем очередь, привязываем её к exchange с указанным routing key
	//    (очередь также durable, autoDelete=false, не внутренняя)
	if err = client.DeclareQueue(cfg.Queue, cfg.Exchange, cfg.RoutingKey, true, false, true, nil); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ошибка объявления очереди: %w", err)
	}

	// 7. Создаём Publisher для отправки сообщений в exchange
	publisher := rabbitmq.NewPublisher(client, cfg.Exchange, "application/json")

	// 8. Канал, через который consumer будет передавать ID просроченных броней сервисному слою
	serviceCh := make(chan int, 100)

	// 9. Запускаем горутину, которая читает ID просрочек из канала, приходящего из ZSet,
	//    и публикует их в RabbitMQ (связывает рэдис и рэббит)
	go startPublishLoop(ctx, publisher, cfg.RoutingKey, overdueCh, log)

	// 10. Определяем обработчик входящих сообщений консумера
	//     (получает Delivery, парсит ID из тела (JSON или просто число) и отправляет в serviceCh)
	handler := func(ctx context.Context, d amqp.Delivery) error {

		var id int
		// пробуем распарсить как JSON-число
		if err := json.Unmarshal(d.Body, &id); err != nil {
			// если не JSON, пробуем преобразовать строку в int
			id, err = strconv.Atoi(string(d.Body))
			if err != nil {
				log.Error("некорректное тело сообщения", "body", string(d.Body))
				// отбрасываем сообщение без повторной постановки в очередь (requeue=false)
				_ = d.Nack(false, false)
				return fmt.Errorf("некорректное тело: %w", err)
			}
		}

		sendTimeout := 1 * time.Second // таймаут отправки в канал сервисного слоя

		// Пытаемся отправить ID в канал сервисного слоя с таймаутом
		select {

		case <-ctx.Done(): // контекст отменён - переотправляем сообщение
			_ = d.Nack(false, true)
			return ctx.Err()

		case serviceCh <- id: // успешно отправили - подтверждаем сообщение
			return nil

		case <-time.After(sendTimeout): // сервисный слой не готов принять сообщение за отведённое время
			log.Error("таймаут отправки ID в сервисный слой", "id", id)
			_ = d.Nack(false, true) // возвращаем в очередь для повторной попытки
			return fmt.Errorf("таймаут отправки в serviceCh")
		}
	}

	// 11. Конфигурация консумера
	consumerCfg := rabbitmq.ConsumerConfig{
		Queue:         cfg.Queue, // имя очереди
		AutoAck:       false,     // без авто-подтверждения
		Ask:           rabbitmq.AskConfig{Multiple: false},
		Nack:          rabbitmq.NackConfig{Multiple: false, Requeue: true},
		Workers:       1,
		PrefetchCount: 1,
	}

	consumer := rabbitmq.NewConsumer(client, consumerCfg, handler)

	// 12. Запускаем consumer в отдельной горутине (при завершении консумера закрываем serviceCh,
	//     чтобы сервисный слой узнал о прекращении поступления данных)
	go func() {
		defer close(serviceCh)
		if err := consumer.Start(ctx); err != nil {
			log.Error("consumer RabbitMQ остановлен с ошибкой", "error", err)
		}
	}()

	// 13. Функция закрытия клиента, которую вернём в структуре Broker для graceful shutdown.
	closeFunc := func() error {
		return client.Close()
	}

	log.Info("RabbitMQ инициализирован", "exchange", cfg.Exchange, "queue", cfg.Queue)

	// 14. Возвращаем экземпляр Broker, содержащий канал для чтения и функцию закрытия.
	return &Broker{
		messages: serviceCh,
		close:    closeFunc,
	}, nil
}

// startPublishLoop читает ID из overdueCh и публикует в RabbitMQ
func startPublishLoop(ctx context.Context, publisher *rabbitmq.Publisher, routingKey string, overdueCh <-chan int, log logger.Logger) {

	for {
		select {

		case <-ctx.Done():
			log.Info("горутина публикации остановлена по контексту")
			return

		case id, ok := <-overdueCh:
			if !ok {
				log.Info("канал просрочек из ZSet закрыт, горутина публикации завершена")
				return
			}
			body, _ := json.Marshal(id)
			if err := publisher.Publish(ctx, body, routingKey); err != nil {
				log.Error("не удалось опубликовать просрочку в RabbitMQ", "id", id, "error", err)
			} else {
				log.Info("просрочка опубликована в RabbitMQ", "id", id)
			}
		}
	}
}
