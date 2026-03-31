// регистрация маршрутов API календаря (CRUD + выборки по дню/неделе/месяцу)
package http

// registerRoutes - привязка путей к обработчикам
func (s *Server) registerRoutes() {

	r := s.engine
	h := s.handler

	// POST /create_event - создание события
	r.POST("/create_event", h.CreateEvent)

	// POST /update_event - обновление события
	r.POST("/update_event", h.UpdateEvent)

	// POST /delete_event - удаление события
	r.POST("/delete_event", h.DeleteEvent)

	// GET /events_for_day - все активные события пользователя на день
	r.GET("/events_for_day", h.EventsForDay)

	// GET /events_for_week - активные события на неделю
	r.GET("/events_for_week", h.EventsForWeek)

	// GET /events_for_month - активные события на месяц
	r.GET("/events_for_month", h.EventsForMonth)

	// GET /archive_events - получение архива пользователя (пагинация)
	r.GET("/archive_events", h.GetArchiveEvents)
}
