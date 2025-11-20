package shutdown

import "sync/atomic"

var isShuttingDown int32 // флаг состояния сервера

// IsShuttingDown проверяет, находится ли приложение в процессе остановки
func IsShuttingDown() bool {

	return atomic.LoadInt32(&isShuttingDown) == 1
}

// StartShutdown помечает приложение как останавливающееся
func StartShutdown() {

	atomic.StoreInt32(&isShuttingDown, 1)
}
