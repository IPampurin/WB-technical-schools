package storage

import (
	"fmt"
	"sync"
	"time"
)

// Repository - интерфейс, реализующий требуемые методы
type Repository interface {
	Create(userID int, date time.Time, title, content string) (int, error) // добавляет event в хранилище, возвращает ID event или ошибку
	Update(event *Event) error                                             // обновляет event в хранилище, возвращает ошибку, если событие не найдено
	Delete(userID, eventID int) error                                      // удаляет event из хранилища, возвращает ошибку, если событие не найдено
	GetForDay(userID int, date time.Time) ([]*Event, error)                // возвращает перечень событий на день или ошибку
	GetForWeek(userID int, date time.Time) ([]*Event, error)               // возвращает перечень событий на неделю или ошибку
	GetForMonth(userID int, date time.Time) ([]*Event, error)              // возвращает перечень событий на месяц или ошибку
}

// Event описывает запись в календаре событий
type Event struct {
	ID      int       `json:"id"`                // id события (счётчик событий)
	UserID  int       `json:"user_id"`           // id пользователя
	Date    time.Time `json:"date"`              // дата события
	Title   string    `json:"title"`             // заголовок события
	Content string    `json:"content,omitempty"` // содержание события
}

// Storage используем для хранения информации календаря событий
type Storage struct {
	Mu     sync.RWMutex     // предполагаем конкурентный доступ к ресурсу
	Events map[int][]*Event // user_id -> events
	NextID int              // номер (ID) следующего Event (счётчик событий)
}

// NewStorage создаёт новое хранилище
func NewStorage() Repository {
	return &Storage{
		Events: make(map[int][]*Event),
		NextID: 1,
	}
}

// Create добавляет event в хранилище, возвращает ID event или ошибку
func (s *Storage) Create(userID int, date time.Time, title, content string) (int, error) {

	// выполняем базовые проверки
	if userID < 0 {
		return 0, fmt.Errorf("ошибочный ID пользователя")
	}
	if title == "" {
		return 0, fmt.Errorf("поле title должно быть заполнено")
	}

	s.Mu.Lock()
	defer s.Mu.Unlock()

	// проверяем, что память под слайс событий есть и пользователь существует
	if _, ok := s.Events[userID]; !ok || s.Events[userID] == nil {
		s.Events[userID] = make([]*Event, 0)
	}

	// добавляем пользователю событие в список
	s.Events[userID] = append(s.Events[userID], &Event{
		ID:      s.NextID,
		UserID:  userID,
		Date:    date,
		Title:   title,
		Content: content,
	})
	// добаляем счётчик событий
	s.NextID++

	return s.NextID - 1, nil
}

// Update обновляет event в хранилище, возвращает ошибку, если событие не найдено
func (s *Storage) Update(event *Event) error {

	if event == nil {
		return fmt.Errorf("событие не может быть nil")
	}

	s.Mu.Lock()
	defer s.Mu.Unlock()

	events, ok := s.Events[event.UserID]
	if !ok {
		return fmt.Errorf("пользователь с %d не найден", event.UserID)
	}
	if events == nil {
		return fmt.Errorf("у пользователя с %d событий ваще не найдено", event.UserID)
	}

	for i := 0; i < len(events); i++ {
		// если нашли событие - обновляем данные
		if event.ID == events[i].ID {
			events[i].Date = event.Date
			events[i].Title = event.Title
			events[i].Content = event.Content
			return nil
		}
	}

	// если событие не найдено - что-то пошло не так
	return fmt.Errorf("событие с %d не найдено", event.ID)
}

// Delete удаляет event из хранилища, возвращает ошибку, если событие не найдено
func (s *Storage) Delete(userID, eventID int) error {

	s.Mu.Lock()
	defer s.Mu.Unlock()

	events, ok := s.Events[userID]
	if !ok {
		return fmt.Errorf("пользователь с %d не найден", userID)
	}
	if events == nil {
		return fmt.Errorf("у пользователя с %d событий ваще не найдено", userID)
	}

	for i := 0; i < len(events); i++ {
		// если нашли событие - удаляем событие
		if eventID == events[i].ID {
			copy(events[i:], events[i+1:])
			s.Events[userID] = events[:len(events)-1]
			// или s.Events[userID] = slices.Delete(s.Events[userID], i, i+1)
			return nil
		}
	}

	// если событие не найдено - что-то пошло не так
	return fmt.Errorf("событие с %d не найдено", eventID)
}

// dayNormalizer возвращает начало дня
func dayNormalizer(t time.Time) time.Time {

	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// возвращает перечень событий на день или ошибку
func (s *Storage) GetForDay(userID int, date time.Time) ([]*Event, error) {

	s.Mu.RLock()
	defer s.Mu.Unlock()

	events, ok := s.Events[userID]
	if !ok {
		return []*Event{}, fmt.Errorf("пользователь с %d не найден", userID)
	}
	if events == nil {
		return []*Event{}, fmt.Errorf("у пользователя с %d событий ваще не найдено", userID)
	}

	eventsForDay := make([]*Event, 0)
	fromDay := dayNormalizer(date)
	toDay := fromDay.AddDate(0, 0, 1)

	for i := 0; i < len(events); i++ {
		if !events[i].Date.Before(fromDay) && events[i].Date.Before(toDay) {
			eventsForDay = append(eventsForDay, events[i])
		}
	}

	return eventsForDay, nil
}

// weekNormalizer возвращает начало недели
func weekNormalizer(t time.Time) time.Time {

	// находим начало дня
	startOfDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	// находим понедельник
	weekday := startOfDay.Weekday()
	if weekday == 0 { // 0 == воскресенье
		return startOfDay.AddDate(0, 0, -6) // начало предыдущего понедельника
	}

	return startOfDay.AddDate(0, 0, -int(weekday)+1)
}

// возвращает перечень событий на неделю или ошибку
func (s *Storage) GetForWeek(userID int, date time.Time) ([]*Event, error) {

	s.Mu.RLock()
	defer s.Mu.Unlock()

	events, ok := s.Events[userID]
	if !ok {
		return []*Event{}, fmt.Errorf("пользователь с %d не найден", userID)
	}
	if events == nil {
		return []*Event{}, fmt.Errorf("у пользователя с %d событий ваще не найдено", userID)
	}

	eventsForWeek := make([]*Event, 0)
	fromDay := weekNormalizer(date)
	toDay := fromDay.AddDate(0, 0, 7)

	for i := 0; i < len(events); i++ {
		if !events[i].Date.Before(fromDay) && events[i].Date.Before(toDay) {
			eventsForWeek = append(eventsForWeek, events[i])
		}
	}

	return eventsForWeek, nil
}

// monthNormalizer возвращает начало месяца
func monthNormalizer(t time.Time) time.Time {

	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// возвращает перечень событий на месяц или ошибку
func (s *Storage) GetForMonth(userID int, date time.Time) ([]*Event, error) {

	s.Mu.RLock()
	defer s.Mu.Unlock()

	events, ok := s.Events[userID]
	if !ok {
		return []*Event{}, fmt.Errorf("пользователь с %d не найден", userID)
	}
	if events == nil {
		return []*Event{}, fmt.Errorf("у пользователя с %d событий ваще не найдено", userID)
	}

	eventsForMonth := make([]*Event, 0)
	fromDay := monthNormalizer(date)
	toDay := fromDay.AddDate(0, 1, 0)

	for i := 0; i < len(events); i++ {
		if !events[i].Date.Before(fromDay) && events[i].Date.Before(toDay) {
			eventsForMonth = append(eventsForMonth, events[i])
		}
	}

	return eventsForMonth, nil
}
