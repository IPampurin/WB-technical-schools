// интерфейс логгера - порт логирования для HTTP, воркеров и сервиса
package service

// Logger - структурированные сообщения (пары key-value как у slog)
type Logger interface {

	// Info - информационное сообщение
	Info(msg string, args ...any)

	// Error - ошибка или сбой (args - пары ключ-значение для контекста)
	Error(msg string, args ...any)

	// Debug - отладка (можно не вызывать в проде или отфильтровать в handler)
	Debug(msg string, args ...any)

	// Close - сброс буфера и остановка фоновой записи, если есть
	Close() error
}
