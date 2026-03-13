package broker

// BrokerMethods - методы любого брокера сообщений в системе
type BrokerMethods interface {

	// Messages предоставляет канал с исходящим данными
	Messages() <-chan int

	// Close предоставляет функцию закрытия брокера сообщений
	Close() error
}
