package overdueworker

import (
	"context"
	"sync"

	"github.com/IPampurin/EventBooker/pkg/broker"
	"github.com/IPampurin/EventBooker/pkg/service"
	"github.com/wb-go/wbf/logger"
)

// Worker обрабатывает просроченные брони из канала брокера
type Worker struct {
	broker  *broker.Broker
	service *service.Service
	log     logger.Logger
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// InitWorker создаёт и запускает воркер
func InitWorker(ctx context.Context, bro *broker.Broker, svc *service.Service, log logger.Logger) *Worker {

	ctx, cancel := context.WithCancel(ctx)

	w := &Worker{
		broker:  bro,
		service: svc,
		log:     log,
		cancel:  cancel,
	}

	w.wg.Add(1)
	go w.run(ctx)

	return w
}

// Stop останавливает воркер и ожидает завершения
func (w *Worker) Stop() {

	w.cancel()
	w.wg.Wait()
}

// run читает из канала брокера и обрабатывает ID броней
func (w *Worker) run(ctx context.Context) {

	defer w.wg.Done()

	ch := w.broker.Messages()

	for {
		select {

		case <-ctx.Done():
			w.log.Info("воркер остановлен по сигналу")
			return

		case id, ok := <-ch:
			if !ok {
				w.log.Info("канал брокера закрыт, воркер завершает работу")
				return
			}
			// вызываем метод сервиса для отмены брони
			if err := w.service.CancelBooking(ctx, id, w.log); err != nil {
				w.log.Error("ошибка отмены просроченной брони", "id", id, "error", err)
			} else {
				w.log.Info("просроченная бронь успешно отменена", "id", id)
			}
		}
	}
}
