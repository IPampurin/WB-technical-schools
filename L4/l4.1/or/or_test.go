package or

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// sig - вспомогательная функция, возвращающая канал, который закроется через заданное время
func sig(after time.Duration) <-chan interface{} {

	c := make(chan interface{})
	go func() {
		defer close(c)
		time.Sleep(after)
	}()

	return c
}

func TestOr(t *testing.T) {

	t.Run("нет каналов", func(t *testing.T) {
		ch := Or()
		_, ok := <-ch
		// ожидаем, что канал закрыт (чтение возвращает zero value и false)
		assert.False(t, ok, "канал должен быть закрыт")
	})

	t.Run("один канал", func(t *testing.T) {
		ch := Or(sig(10 * time.Millisecond))
		start := time.Now()
		<-ch
		// проверяем, что закрытие произошло не раньше, чем через 10 мс
		assert.True(t, time.Since(start) >= 10*time.Millisecond, "возвращённый канал закрылся слишком рано")
	})

	t.Run("несколько каналов", func(t *testing.T) {
		start := time.Now()
		ch := Or(
			sig(50*time.Millisecond),
			sig(20*time.Millisecond),
			sig(100*time.Millisecond),
		)
		<-ch
		elapsed := time.Since(start)
		// ожидаем закрытие через ~20 мс (с небольшим допуском)
		assert.True(t, elapsed >= 20*time.Millisecond && elapsed <= 30*time.Millisecond, "ожидалось закрытие через ~20мс, получено %v", elapsed)
	})

	t.Run("уже закрытый канал", func(t *testing.T) {
		closed := make(chan interface{})
		close(closed)
		ch := Or(closed, sig(1*time.Second))
		start := time.Now()
		<-ch
		// канал закрыт сразу, время должно быть близко к нулю
		assert.True(t, time.Since(start) <= 10*time.Millisecond, "не закрылся сразу при наличии закрытого канала")
	})

	t.Run("нет утечек горутин", func(t *testing.T) {

		// сохраняем текущее количество горутин до запуска
		initialGoroutines := runtime.NumGoroutine()

		// запускаем Or с несколькими каналами
		ch := Or(
			sig(10*time.Millisecond),
			sig(20*time.Millisecond),
			sig(30*time.Millisecond),
		)

		// ждём закрытия выходного канала
		<-ch

		// даём время всем горутинам завершиться
		time.Sleep(50 * time.Millisecond)

		// проверяем, что количество горутин вернулось к исходному
		finalGoroutines := runtime.NumGoroutine()
		assert.LessOrEqual(t, finalGoroutines, initialGoroutines, "обнаружены утечки горутин: было %d, стало %d", initialGoroutines, finalGoroutines)
	})
}
