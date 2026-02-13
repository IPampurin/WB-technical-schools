package api

import (
	"net/http"

	"github.com/IPampurin/calendar-server/pkg/storage"
)

type API struct {
	Storage storage.Repository
}

func NewAPI(db storage.Repository) *API {
	return &API{Storage: db}
}

func Init(db storage.Repository) {

	api := NewAPI(db)

	http.HandleFunc("POST /create_event", api.CreateEventHandler)          // POST — создание нового события
	http.HandleFunc("POST /update_event", api.UpdateEventHandler)          // POST — обновление существующего
	http.HandleFunc("POST /delete_event", api.DeleteEventHandler)          // POST — удаление
	http.HandleFunc("GET /events_for_day", api.GetEventsForDayHandler)     // GET — события на день
	http.HandleFunc("GET /events_for_week", api.GetEventsForWeekHandler)   // GET — события на неделю
	http.HandleFunc("GET /events_for_month", api.GetEventsForMonthHandler) // GET — события на месяц
}
