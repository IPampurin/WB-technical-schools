package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/IPampurin/WarehouseControlAntiPattern/pkg/auth"
	"github.com/IPampurin/WarehouseControlAntiPattern/pkg/domain"
	"github.com/IPampurin/WarehouseControlAntiPattern/pkg/service"
	"github.com/gin-gonic/gin"
	"github.com/wb-go/wbf/logger"
)

// abortWithError отправляет JSON-ответ с ошибкой и завершает запрос
func abortWithError(c *gin.Context, err error) {

	switch {
	case errors.Is(err, domain.ErrNotFound):
		c.AbortWithStatusJSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
	case errors.Is(err, domain.ErrForbidden):
		c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{Error: err.Error()})
	case errors.Is(err, domain.ErrInvalidInput):
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	default:
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "внутренняя ошибка сервера"})
	}
}

// товары

// CreateItem обрабатывает POST /items
func CreateItem(svc service.ServiceMethods, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		req := &CreateUpdateItemRequest{}
		if err := c.ShouldBindJSON(req); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
			return
		}

		item := itemFromCreateRequest(req)
		created, err := svc.CreateItem(c.Request.Context(), item)
		if err != nil {
			abortWithError(c, err)
			return
		}

		c.JSON(http.StatusCreated, itemToAPI(created))
	}
}

// GetItems обрабатывает GET /items
func GetItems(svc service.ServiceMethods, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		items, err := svc.GetAllItems(c.Request.Context())
		if err != nil {
			abortWithError(c, err)
			return
		}

		resp := make([]*Item, 0, len(items))
		for _, item := range items {
			resp = append(resp, itemToAPI(item))
		}

		c.JSON(http.StatusOK, resp)
	}
}

// UpdateItem обрабатывает PUT /items/:id
func UpdateItem(svc service.ServiceMethods, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "ошибка ID товара"})
			return
		}

		req := &CreateUpdateItemRequest{}
		if err := c.ShouldBindJSON(req); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
			return
		}

		item := itemFromUpdateRequest(id, req)
		updated, err := svc.UpdateItem(c.Request.Context(), item)
		if err != nil {
			abortWithError(c, err)
			return
		}

		c.JSON(http.StatusOK, itemToAPI(updated))
	}
}

// DeleteItem обрабатывает DELETE /items/:id
func DeleteItem(svc service.ServiceMethods, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "ошибка ID товара"})
			return
		}

		if err := svc.DeleteItem(c.Request.Context(), id); err != nil {
			abortWithError(c, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

// история

// GetItemHistory обрабатывает GET /items/:id/history
func GetItemHistory(svc service.ServiceMethods, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		idStr := c.Param("id")
		itemID, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "ошибка ID товара"})
			return
		}

		query := &HistoryQuery{}
		if err := c.ShouldBindQuery(query); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
			return
		}

		fromDate, err := parseTimeSafe(query.FromDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "некорректный формат from_date"})
			return
		}
		toDate, err := parseTimeSafe(query.ToDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "некорректный формат to_date"})
			return
		}

		filter := &domain.HistoryFilter{
			FromDate: fromDate,
			ToDate:   toDate,
			UserID:   query.UserID,
			Action:   query.Action,
			Limit:    query.Limit,
			Offset:   query.Offset,
		}

		// itemID передаётся отдельным параметром, в сервисе он будет установлен принудительно

		history, err := svc.GetItemHistory(c.Request.Context(), itemID, filter)
		if err != nil {
			abortWithError(c, err)
			return
		}

		resp := make([]*HistoryRecord, 0, len(history))
		for _, h := range history {
			resp = append(resp, historyToAPI(h))
		}

		c.JSON(http.StatusOK, resp)
	}
}

// GetGlobalHistory обрабатывает GET /history
func GetGlobalHistory(svc service.ServiceMethods, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		query := &HistoryQuery{}
		if err := c.ShouldBindQuery(query); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
			return
		}

		fromDate, err := parseTimeSafe(query.FromDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "некорректный формат from_date"})
			return
		}
		toDate, err := parseTimeSafe(query.ToDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "некорректный формат to_date"})
			return
		}

		filter := &domain.HistoryFilter{
			ItemID:   query.ItemID,
			FromDate: fromDate,
			ToDate:   toDate,
			UserID:   query.UserID,
			Action:   query.Action,
			Limit:    query.Limit,
			Offset:   query.Offset,
		}

		history, err := svc.GetGlobalHistory(c.Request.Context(), filter)
		if err != nil {
			abortWithError(c, err)
			return
		}

		resp := make([]*HistoryRecord, 0, len(history))
		for _, h := range history {
			resp = append(resp, historyToAPI(h))
		}

		c.JSON(http.StatusOK, resp)
	}
}

// CompareHistory обрабатывает GET /history/compare
func CompareHistory(svc service.ServiceMethods, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		fromStr := c.Query("from")
		toStr := c.Query("to")
		if fromStr == "" || toStr == "" {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "параметры 'from' и 'to' обязательны"})
			return
		}

		fromID, err := strconv.Atoi(fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "некорректный параметр 'from'"})
			return
		}
		toID, err := strconv.Atoi(toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "некорректный параметр 'to'"})
			return
		}

		diff, err := svc.CompareHistoryVersions(c.Request.Context(), fromID, toID)
		if err != nil {
			abortWithError(c, err)
			return
		}

		c.JSON(http.StatusOK, diff)
	}
}

// экспорт

// ExportItemsCSV обрабатывает GET /export/items/csv
func ExportItemsCSV(svc service.ServiceMethods, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		data, err := svc.ExportItemsCSV(c.Request.Context())
		if err != nil {
			abortWithError(c, err)
			return
		}

		c.Header("Content-Disposition", "attachment; filename=items.csv")
		c.Data(http.StatusOK, "text/csv", data)
	}
}

// ExportHistoryCSV обрабатывает GET /export/history/csv
func ExportHistoryCSV(svc service.ServiceMethods, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		query := &ExportHistoryQuery{}
		if err := c.ShouldBindQuery(query); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
			return
		}

		fromDate, err := parseTimeSafe(query.FromDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "некорректный формат from_date"})
			return
		}
		toDate, err := parseTimeSafe(query.ToDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "некорректный формат to_date"})
			return
		}

		filter := &domain.HistoryFilter{
			ItemID:   query.ItemID,
			FromDate: fromDate,
			ToDate:   toDate,
			UserID:   query.UserID,
			Action:   query.Action,
		}

		data, err := svc.ExportHistoryCSV(c.Request.Context(), filter)
		if err != nil {
			abortWithError(c, err)
			return
		}

		c.Header("Content-Disposition", "attachment; filename=history.csv")
		c.Data(http.StatusOK, "text/csv", data)
	}
}

// Login обрабатывает POST /login
// (назначает фиксированный userID в зависимости от роли)
// (в реальном проекте здесь должна быть проверка учетных данных в базе данных)
func Login(c *gin.Context) {

	req := &LoginRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "некорректные данные: " + err.Error()})
		return
	}

	// фиксированные id для упрощения
	var userID int
	switch req.Role {
	case "admin":
		userID = 1
	case "manager":
		userID = 2
	case "viewer":
		userID = 3
	}

	token, err := auth.GenerateToken(userID, req.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "ошибка генерации токена"})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{Token: token})
}
