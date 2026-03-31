// логгер
package logger

import (
	"fmt"
	"sync"
	"time"
)

// Entry - одна запись лога
type Entry struct {
	Level   string
	Time    time.Time
	Message string
	Fields  map[string]interface{}
}

// AsyncLogger - асинхронный логгер, пишущий в stdout
type AsyncLogger struct {
	ch   chan Entry
	done chan struct{}
	wg   sync.WaitGroup
}

// NewAsyncLogger создаёт логгер с буфером size
func NewAsyncLogger(size int) *AsyncLogger {

	l := &AsyncLogger{
		ch:   make(chan Entry, size),
		done: make(chan struct{}),
	}

	l.wg.Add(1)
	go l.process()

	return l
}

// process - фоновая горутина, читающая из канала
func (l *AsyncLogger) process() {

	defer l.wg.Done()

	for entry := range l.ch {
		// простой вывод в stdout
		fmt.Printf("%s [%s] %s %v\n",
			entry.Time.Format(time.RFC3339),
			entry.Level,
			entry.Message,
			entry.Fields,
		)
	}

	close(l.done)
}

// send - внутренняя отправка записи
func (l *AsyncLogger) send(level, msg string, fields ...interface{}) {

	m := make(map[string]interface{})
	for i := 0; i+1 < len(fields); i += 2 {
		if key, ok := fields[i].(string); ok {
			m[key] = fields[i+1]
		}
	}

	select {
	case l.ch <- Entry{Level: level, Time: time.Now(), Message: msg, Fields: m}:
	default:
		// канал переполнен - теряем лог (можно добавить счётчик)
	}
}

func (l *AsyncLogger) Info(msg string, fields ...interface{}) {
	l.send("INFO", msg, fields...)
}

func (l *AsyncLogger) Error(msg string, fields ...interface{}) {
	l.send("ERROR", msg, fields...)
}

func (l *AsyncLogger) Debug(msg string, fields ...interface{}) {
	l.send("DEBUG", msg, fields...)
}

// Close - закрывает канал и ждёт завершения писателя
func (l *AsyncLogger) Close() error {

	close(l.ch)
	<-l.done
	l.wg.Wait()

	return nil
}
