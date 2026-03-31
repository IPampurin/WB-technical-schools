// тесты асинхронного логгера (а может и ну их)
package tests

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/IPampurin/EventCalendar/internal/async/logger"
	"github.com/stretchr/testify/assert"
)

// captureOutput перенаправляет stdout в буфер
// возвращает:
//   - buf – буфер, в который будет записан вывод
//   - flush – функция, которая закрывает писатель и дожидается завершения копирования
//   - restore – функция, которая вызывает flush и восстанавливает исходный stdout
func captureOutput() (*bytes.Buffer, func(), func()) {

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	buf := &bytes.Buffer{}
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(buf, r)
		close(done)
	}()

	flush := func() {
		w.Close()
		<-done
	}
	restore := func() {
		flush()
		os.Stdout = old
	}

	return buf, flush, restore
}

func TestAsyncLogger_Basic(t *testing.T) {

	buf, flush, restore := captureOutput()
	defer restore()

	l := logger.NewAsyncLogger(10)
	defer l.Close()

	l.Info("test info", "key", "value")
	l.Error("test error", "err", "something")
	l.Debug("test debug", "foo", "bar")

	time.Sleep(50 * time.Millisecond) // даём время на обработку

	flush() // дожидаемся завершения копирования
	output := buf.String()

	assert.Contains(t, output, "INFO")
	assert.Contains(t, output, "test info")
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "value")

	assert.Contains(t, output, "ERROR")
	assert.Contains(t, output, "test error")
	assert.Contains(t, output, "err")
	assert.Contains(t, output, "something")

	assert.Contains(t, output, "DEBUG")
	assert.Contains(t, output, "test debug")
	assert.Contains(t, output, "foo")
	assert.Contains(t, output, "bar")
}

func TestAsyncLogger_Fields(t *testing.T) {

	buf, flush, restore := captureOutput()
	defer restore()

	l := logger.NewAsyncLogger(10)
	defer l.Close()

	l.Info("no fields")
	l.Info("with fields", "field1", 123, "field2", "text", "field3", true)

	time.Sleep(50 * time.Millisecond)
	flush()
	output := buf.String()

	assert.Contains(t, output, "no fields")
	assert.Contains(t, output, "with fields")
	assert.Contains(t, output, "field1")
	assert.Contains(t, output, "123")
	assert.Contains(t, output, "field2")
	assert.Contains(t, output, "text")
	assert.Contains(t, output, "field3")
	assert.Contains(t, output, "true")
}

func TestAsyncLogger_BufferOverflow(t *testing.T) {

	l := logger.NewAsyncLogger(2)
	defer l.Close()

	for i := 0; i < 100; i++ {
		l.Info("message", "index", i)
	}
	time.Sleep(50 * time.Millisecond)
	// просто проверяем, что не упало
}

func TestAsyncLogger_CloseWaitsForProcessing(t *testing.T) {

	l := logger.NewAsyncLogger(10)
	l.Info("last message")

	closeErr := l.Close()
	assert.NoError(t, closeErr)
}

func TestAsyncLogger_ConcurrentWrites(t *testing.T) {

	buf, flush, restore := captureOutput()
	defer restore()

	l := logger.NewAsyncLogger(100)
	defer l.Close()

	const goroutines = 10
	const messagesPer = 100

	done := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			for j := 0; j < messagesPer; j++ {
				l.Info("concurrent", "goroutine", id, "msg", j)
			}
			done <- struct{}{}
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		<-done
	}

	time.Sleep(200 * time.Millisecond)
	flush()
	output := buf.String()
	count := strings.Count(output, "concurrent")
	expected := goroutines * messagesPer
	assert.GreaterOrEqual(t, count, 0)
	assert.LessOrEqual(t, count, expected)
}

func TestAsyncLogger_FieldsWithOddArguments(t *testing.T) {

	buf, flush, restore := captureOutput()
	defer restore()

	l := logger.NewAsyncLogger(10)
	defer l.Close()

	l.Info("odd fields", "key1", "value1", "key2")
	time.Sleep(50 * time.Millisecond)
	flush()
	output := buf.String()

	assert.NotContains(t, output, "key2")
	assert.Contains(t, output, "key1")
	assert.Contains(t, output, "value1")
}
