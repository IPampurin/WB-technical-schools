package tests

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/IPampurin/calendar-server/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewStorage проверяет создание нового хранилища
func TestNewStorage(t *testing.T) {

	s := storage.NewStorage()

	require.NotNil(t, s.Events, "Events map не должен быть nil")
	require.Equal(t, 1, s.NextID, "NextID должен быть 1")
}

// TestCreate проверяет создание событий
func TestCreate(t *testing.T) {

	s := storage.NewStorage()
	date := time.Now()

	// создаем первое событие - ожидаем ID = 1
	t.Log("Создание первого события")
	id, err := s.Create(1, date, "Meeting", "Team meeting")
	assert.NoError(t, err, "Создание события не должно вернуть ошибку")
	assert.Equal(t, 1, id, "ID первого события должен быть 1")

	// создаем второе событие для того же пользователя - ожидаем ID = 2
	t.Log("Создание второго события для того же пользователя")
	id2, err := s.Create(1, date, "Lunch", "With colleagues")
	assert.NoError(t, err, "Создание второго события не должно вернуть ошибку")
	assert.Equal(t, 2, id2, "ID второго события должен быть 2 (инкремент счетчика)")

	// попытка создать событие с пустым заголовком
	t.Log("Проверка валидации: пустой заголовок")
	_, err = s.Create(1, date, "", "Empty title")
	assert.Error(t, err, "Должна быть ошибка при пустом заголовке")
	assert.Contains(t, err.Error(), "title", "Ошибка должна упоминать поле title")

	// попытка создать событие с отрицательным ID пользователя
	t.Log("Проверка валидации: отрицательный userID")
	_, err = s.Create(-1, date, "Test", "Test")
	assert.Error(t, err, "Должна быть ошибка при отрицательном userID")
	assert.Contains(t, err.Error(), "ошибочный ID", "Ошибка должна указывать на некорректный ID")
}

// TestUpdate проверяет обновление существующего события
func TestUpdate(t *testing.T) {

	s := storage.NewStorage()
	date := time.Now()

	// создаем событие для последующего обновления
	id, err := s.Create(1, date, "Old title", "Old content")
	require.NoError(t, err, "Не удалось создать тестовое событие")
	require.Equal(t, 1, id, "ID тестового события должен быть 1")

	// готовим обновленные данные: меняем дату (+1 день), заголовок и содержание
	newDate := date.Add(24 * time.Hour)
	updatedEvent := &storage.Event{
		ID:      id, // ID должен совпадать с существующим событием
		UserID:  1,  // UserID должен совпадать с владельцем
		Date:    newDate,
		Title:   "New title",
		Content: "New content",
	}

	// обновляем событие
	err = s.Update(updatedEvent)
	assert.NoError(t, err, "Обновление существующего события не должно вернуть ошибку")

	// получаем событие на новую дату и проверяем поля
	events, err := s.GetForDay(1, newDate)
	require.NoError(t, err, "Не удалось получить события на обновленную дату")
	require.Len(t, events, 1, "На новую дату должно быть ровно одно событие")

	// проверяем, что все поля действительно обновились
	assert.Equal(t, "New title", events[0].Title, "Заголовок не обновился")
	assert.Equal(t, "New content", events[0].Content, "Содержание не обновилось")
	assert.True(t, events[0].Date.Equal(newDate), "Дата не обновилась")

	// попытка обновить несуществующее событие
	t.Log("Проверка: обновление несуществующего события")
	err = s.Update(&storage.Event{ID: 999, UserID: 1})
	assert.Error(t, err, "Должна быть ошибка при обновлении несуществующего события")
	assert.Contains(t, err.Error(), "не найдено", "Ошибка должна указывать, что событие не найдено")

	// передача nil вместо указателя на событие
	t.Log("Проверка: передача nil")
	err = s.Update(nil)
	assert.Error(t, err, "Должна быть ошибка при передаче nil")
	assert.Contains(t, err.Error(), "nil", "Ошибка должна упоминать nil")
}

// TestDelete проверяет удаление событий
func TestDelete(t *testing.T) {

	s := storage.NewStorage()
	date := time.Now()

	// создаем событие для удаления
	id, err := s.Create(1, date, "To delete", "Content")
	require.NoError(t, err, "Не удалось создать тестовое событие")
	require.Equal(t, 1, id, "ID тестового события должен быть 1")

	// удаляем событие
	err = s.Delete(1, id)
	assert.NoError(t, err, "Удаление существующего события не должно вернуть ошибку")

	// убеждаемся, что событие действительно удалено
	events, err := s.GetForDay(1, date)
	require.NoError(t, err, "Не удалось получить список событий")
	assert.Empty(t, events, "После удаления список событий должен быть пуст")

	// попытка удалить уже удаленное (несуществующее) событие
	t.Log("Проверка: удаление несуществующего события")
	err = s.Delete(1, 999)
	assert.Error(t, err, "Должна быть ошибка при удалении несуществующего события")
	assert.Contains(t, err.Error(), "не найдено", "Ошибка должна указывать, что событие не найдено")

	// попытка удалить событие у несуществующего пользователя
	t.Log("Проверка: удаление у несуществующего пользователя")
	err = s.Delete(999, 1)
	assert.Error(t, err, "Должна быть ошибка при удалении у несуществующего пользователя")
	assert.Contains(t, err.Error(), "не найден", "Ошибка должна указывать, что пользователь не найден")
}

// TestGetForDay проверяет получение событий за конкретный день
func TestGetForDay(t *testing.T) {

	s := storage.NewStorage()

	// фиксированная дата для предсказуемости теста
	baseDate := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	// создаем события на разные дни
	// событие в целевой день (15.01.2024)
	_, err := s.Create(1, baseDate, "Event 1", "")
	require.NoError(t, err, "Не удалось создать событие на целевую дату")

	// событие на следующий день (16.01.2024) - не должно попасть в выборку
	_, err = s.Create(1, baseDate.Add(24*time.Hour), "Event 2", "")
	require.NoError(t, err, "Не удалось создать событие на следующий день")

	// событие на предыдущий день (14.01.2024) - не должно попасть в выборку
	_, err = s.Create(1, baseDate.Add(-24*time.Hour), "Event 3", "")
	require.NoError(t, err, "Не удалось создать событие на предыдущий день")

	// получение событий за день
	t.Log("Получение событий за 15.01.2024")
	events, err := s.GetForDay(1, baseDate)
	assert.NoError(t, err, "Получение событий не должно вернуть ошибку")
	assert.Len(t, events, 1, "Должно быть ровно одно событие за целевую дату")
	assert.Equal(t, "Event 1", events[0].Title, "Найдено не то событие")

	// запрашиваем день, на который нет событий
	t.Log("Получение событий за день без событий")
	emptyDay := baseDate.Add(48 * time.Hour) // 17.01.2024
	events, err = s.GetForDay(1, emptyDay)
	assert.NoError(t, err, "Получение пустого списка не должно вернуть ошибку")
	assert.Empty(t, events, "Должен вернуться пустой слайс, а не nil")

	// проверяем несуществующего пользователя
	t.Log("Проверка: получение событий несуществующего пользователя")
	events, err = s.GetForDay(999, baseDate)
	assert.Error(t, err, "Должна быть ошибка для несуществующего пользователя")
	assert.Contains(t, err.Error(), "не найден", "Ошибка должна указывать, что пользователь не найден")
	assert.Empty(t, events, "При ошибке должен возвращаться пустой слайс")
}

// TestGetForWeek проверяет получение событий за неделю
func TestGetForWeek(t *testing.T) {

	s := storage.NewStorage()

	// 15 января 2024 - понедельник
	monday := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	// создаем события внутри и вне целевой недели
	// событие в понедельник - должно быть в выборке
	_, err := s.Create(1, monday, "Monday event", "")
	require.NoError(t, err)

	// событие в среду - должно быть в выборке
	_, err = s.Create(1, monday.Add(48*time.Hour), "Wednesday event", "")
	require.NoError(t, err)

	// событие в следующий понедельник - не должно быть в выборке
	_, err = s.Create(1, monday.Add(7*24*time.Hour), "Next Monday", "")
	require.NoError(t, err)

	// получение событий за неделю
	t.Log("Получение событий за неделю с 15.01.2024")
	events, err := s.GetForWeek(1, monday)
	assert.NoError(t, err, "Получение событий за неделю не должно вернуть ошибку")
	assert.Len(t, events, 2, "Должно быть ровно 2 события (пн и ср)")

	// проверяем, что это именно те события, которые мы ожидаем (проверяем заголовки)
	titles := []string{events[0].Title, events[1].Title}
	assert.Contains(t, titles, "Monday event", "Событие понедельника не найдено")
	assert.Contains(t, titles, "Wednesday event", "Событие среды не найдено")
}

// TestGetForMonth проверяет получение событий за месяц
func TestGetForMonth(t *testing.T) {

	s := storage.NewStorage()

	// 15 января 2024 - середина месяца
	january := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	// создаем события в январе и феврале
	// событие в январе - должно быть в выборке
	_, err := s.Create(1, january, "January event", "")
	require.NoError(t, err)

	// событие 15 февраля (январь + 31 день) - не должно быть в выборке за январь
	_, err = s.Create(1, january.AddDate(0, 1, 0), "February event", "")
	require.NoError(t, err)

	// получение событий за январь
	t.Log("Получение событий за январь 2024")
	events, err := s.GetForMonth(1, january)
	assert.NoError(t, err, "Получение событий за месяц не должно вернуть ошибку")
	assert.Len(t, events, 1, "Должно быть ровно одно событие в январе")
	assert.Equal(t, "January event", events[0].Title, "Найдено не то событие")

	// проверка границ месяца
	// проверяем последний день января
	lastDayOfJanuary := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
	events, err = s.GetForMonth(1, lastDayOfJanuary)
	assert.NoError(t, err)
	assert.Len(t, events, 1, "Даже при запросе в последний день января должно найтись событие")
}

// TestConcurrentAccess проверяет, что хранилище корректно работает при конкурентном доступе
func TestConcurrentAccess(t *testing.T) {

	s := storage.NewStorage()
	date := time.Now()

	// количество конкурентных операций
	const goroutinesCount = 100

	var wg sync.WaitGroup
	wg.Add(goroutinesCount)

	// канал для сбора ошибок от горутин
	// используем буферизированный канал, чтобы горутины не блокировались при отправке
	errors := make(chan error, goroutinesCount)

	// запускаем конкурентное создание событий
	t.Logf("Запуск %d горутин для конкурентного создания событий", goroutinesCount)
	for i := 0; i < goroutinesCount; i++ {
		go func(id int) {
			defer wg.Done()
			// каждая горутина создает свое событие
			_, err := s.Create(1, date, fmt.Sprintf("Concurrent Event %d", id), "")
			if err != nil {
				errors <- err
			}
		}(i)
	}

	// ожидаем завершения всех горутин
	wg.Wait()
	close(errors)

	// проверяем, что не было ошибок при создании
	for err := range errors {
		assert.NoError(t, err, "Горутина вернула ошибку при создании")
	}

	// проверяем, что все 100 событий действительно создались
	events, err := s.GetForDay(1, date)
	assert.NoError(t, err, "Не удалось получить список событий")
	assert.Len(t, events, goroutinesCount, "Должно быть создано ровно %d событий, но создано %d", goroutinesCount, len(events))

	// проверка, что ID событий уникальные
	// (это косвенно проверяет, что счетчик NextID увеличивается атомарно)
	ids := make(map[int]bool)
	for _, event := range events {
		assert.False(t, ids[event.ID], "Обнаружен дубликат ID: %d", event.ID)
		ids[event.ID] = true
	}
}
