package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/IPampurin/EventBooker/pkg/service"
	"github.com/gin-gonic/gin"
	"github.com/wb-go/wbf/logger"
)

// CreateEvent возвращает обработчик для POST /events (создание мероприятия)
func CreateEvent(svc *service.Service, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		var req createEventRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "некорректные данные: " + err.Error()})
			return
		}

		id, err := svc.EventCreater(c.Request.Context(), req.Name, req.Date, req.BookingTTLMinutes, req.TotalSeats, req.BookingPrice, log)
		if err != nil {
			log.Error("ошибка создания мероприятия", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось создать мероприятие"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"id": id})
	}
}

// ReserveSeat возвращает обработчик для POST /events/:id/book (бронирование места)
func ReserveSeat(svc *service.Service, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		eventID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный ID мероприятия"})
			return
		}

		var req bookRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "некорректные данные: " + err.Error()})
			return
		}

		createdAt := time.Now()

		bookingID, err := svc.SeatReserver(c.Request.Context(), eventID, req.UserID, createdAt, log)
		if err != nil {
			log.Error("ошибка бронирования", "event_id", eventID, "user_id", req.UserID, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось забронировать место"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"bookingId": bookingID})
	}
}

// ConfirmReserve возвращает обработчик для POST /events/:id/confirm (оплата брони (если мероприятие требует этого))
func ConfirmReserve(svc *service.Service, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		var req confirmRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "некорректные данные: " + err.Error()})
			return
		}

		if err := svc.ReserveConfirmer(c.Request.Context(), req.BookingID, log); err != nil {
			log.Error("ошибка подтверждения брони", "booking_id", req.BookingID, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось подтвердить бронь"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "confirmed"})
	}
}

// GetEventByID возвращает обработчик для GET /events/:id (получение информации о мероприятии и свободных местах)
func GetEventByID(svc *service.Service, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный ID мероприятия"})
			return
		}

		event, err := svc.GetEventByID(c.Request.Context(), id, log)
		if err != nil {
			log.Error("ошибка получения мероприятия", "id", id, "error", err)
			c.JSON(http.StatusNotFound, gin.H{"error": "мероприятие не найдено"})
			return
		}

		// преобразуем
		response := &EventResponse{
			ID:                event.ID,
			Name:              event.Name,
			DateEvent:         event.DateEvent,
			BookingTTLMinutes: event.BookingTTLMinutes,
			TotalSeats:        event.TotalSeats,
			FreeSeats:         event.FreeSeats,
			BookedSeats:       event.BookedSeats,
			BookingPrice:      event.BookingPrice,
		}

		c.JSON(http.StatusOK, response)
	}
}

// GetAllEvents возвращает обработчик для GET /events (список всех предстоящих мероприятий)
func GetAllEvents(svc *service.Service, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		events, err := svc.GetEvents(c.Request.Context(), log)
		if err != nil {
			log.Error("ошибка получения списка мероприятий", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось получить мероприятия"})
			return
		}

		// преобразуем
		response := make([]*EventResponse, len(events))
		for i := range events {
			response[i] = &EventResponse{
				ID:                events[i].ID,
				Name:              events[i].Name,
				DateEvent:         events[i].DateEvent,
				BookingTTLMinutes: events[i].BookingTTLMinutes,
				TotalSeats:        events[i].TotalSeats,
				FreeSeats:         events[i].FreeSeats,
				BookedSeats:       events[i].BookedSeats,
				BookingPrice:      events[i].BookingPrice,
			}
		}

		c.JSON(http.StatusOK, response)
	}
}

// RegisterUser возвращает обработчик для POST /register (регистрация пользователя)
func RegisterUser(svc *service.Service, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		var req registerRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "некорректные данные: " + err.Error()})
			return
		}

		id, err := svc.RegisterUser(c.Request.Context(), req.Name, req.Email, log)
		if err != nil {
			log.Error("ошибка регистрации пользователя", "email", req.Email, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось зарегистрировать пользователя"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"id": id})
	}
}

// LoginUser обрабатывает POST /login
func LoginUser(svc *service.Service, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		var req loginRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный email"})
			return
		}

		id, err := svc.LoginUser(c.Request.Context(), req.Email, log)
		if err != nil {
			log.Error("ошибка входа", "email", req.Email, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка"})
			return
		}

		if id == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "пользователь не найден"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"id": id})
	}
}

// GetUserBooking обрабатывает GET /events/:id/book?user_id=
func GetUserBooking(svc *service.Service, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		eventID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный ID мероприятия"})
			return
		}

		userID, err := strconv.Atoi(c.Query("user_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный user_id"})
			return
		}

		bookingID, status, err := svc.GetEventReserveOfUser(c.Request.Context(), eventID, userID, log)
		if err != nil {
			log.Error("ошибка получения брони", "event", eventID, "user", userID, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка"})
			return
		}

		if bookingID == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "бронь не найдена"})
			return
		}

		c.JSON(http.StatusOK, userBookingResponse{
			BookingID: bookingID,
			Status:    status,
		})
	}
}
