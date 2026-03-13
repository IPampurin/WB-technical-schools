package api

import "time"

// createEventRequest тело запроса для создания мероприятия
type createEventRequest struct {
	Name              string    `json:"name" binding:"required"`
	Date              time.Time `json:"date" binding:"required"`
	BookingTTLMinutes int       `json:"bookingTTLMinutes" binding:"required,min=1"`
	TotalSeats        int       `json:"totalSeats" binding:"required,min=1"`
	BookingPrice      int       `json:"bookingPrice" binding:"min=0"`
}

// confirmRequest тело запроса для подтверждения брони
type confirmRequest struct {
	BookingID int `json:"bookingId" binding:"required"`
}

// bookRequest тело запроса для бронирования
type bookRequest struct {
	UserID int `json:"userId" binding:"required"`
}

// registerRequest тело запроса для регистрации пользователя
type registerRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

// loginRequest тело запроса для входа
type loginRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// EventResponse — данные мероприятия для клиента
type EventResponse struct {
	ID                int       `json:"id"`
	Name              string    `json:"name"`
	DateEvent         time.Time `json:"dateEvent"`
	BookingTTLMinutes int       `json:"bookingTTLMinutes"`
	TotalSeats        int       `json:"totalSeats"`
	FreeSeats         int       `json:"freeSeats"`
	BookedSeats       int       `json:"bookedSeats"`
	BookingPrice      int       `json:"bookingPrice"`
}

// userBookingResponse — ответ на запрос GET /events/:id/book?user_id=
type userBookingResponse struct {
	BookingID int    `json:"bookingId"`
	Status    string `json:"status"`
}
