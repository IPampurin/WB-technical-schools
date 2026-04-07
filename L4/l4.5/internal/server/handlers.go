package server

import (
	"net/http"

	"github.com/IPampurin/pprofSimpleService/internal/interfaces"
	"github.com/gin-gonic/gin"
)

// handler реализует методы-обработчики HTTP-запросов
type handler struct {
	svc interfaces.ServiceMethods
}

// Sum обрабатывает GET /sum
func (h *handler) Sum(c *gin.Context) {

	var req SumRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "неверные параметры"})
		return
	}

	if req.A == nil || req.B == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "параметры a и b обязательны"})
		return
	}

	result := h.svc.Sum(*req.A, *req.B)

	c.JSON(http.StatusOK, SuccessResponse{Result: result})
}

// Fib обрабатывает GET /fib
func (h *handler) Fib(c *gin.Context) {

	var req FibRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "параметр n обязателен, целое число от 1 до 45"})
		return
	}

	result := h.svc.Fib(req.N)

	c.JSON(http.StatusOK, SuccessResponse{Result: result})
}

// Allocate обрабатывает POST /allocate
func (h *handler) Allocate(c *gin.Context) {

	var req AllocateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "неверный JSON: требуется поле size (целое положительное число)"})
		return
	}

	allocated, err := h.svc.Allocate(req.Size)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Result: map[string]int{"allocated_bytes": allocated}})
}
