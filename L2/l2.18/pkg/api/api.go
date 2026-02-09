package api

import (
	"net/http"

	"github.com/IPampurin/calendar-server/pkg/storage"
)

func Init(db storage.Repository) {

	http.HandleFunc("/create_event", CreateEventHandler) // POST — создание нового события

	http.HandleFunc("/update_event", UpdateEventHandler) // POST — обновление существующего

	http.HandleFunc("/delete_event", DeleteEventHandler) // POST — удаление

	http.HandleFunc("/events_for_day", GetEventsForDayHandler) // GET — получить все события на день

	http.HandleFunc("/events_for_week", GetEventsForWeekHandler) // GET — события на неделю

	http.HandleFunc("/events_for_month", GetEventsForMonthHandler) // GET — события на месяц
}

// Answer - ответ на запрос к календарю
type Answer struct {
	ID    string `json:"id,omitempty"`
	Error string `json:"error,omitempty"`
}
