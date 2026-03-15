package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/IPampurin/SalesTracker/pkg/domain"
	"github.com/IPampurin/SalesTracker/pkg/service"
	"github.com/wb-go/wbf/ginext"
	"github.com/wb-go/wbf/logger"
)

// GetItemsPeriodSorted возвращает список записей за период с сортировкой
func GetItemsPeriodSorted(svc *service.Service, log logger.Logger) ginext.HandlerFunc {
	return func(c *ginext.Context) {

		query := ItemsQuery{}
		if err := c.ShouldBindQuery(&query); err != nil {
			log.Error("GetItemsPeriodSorted: ошибка парсинга query", "error", err)
			c.JSON(http.StatusBadRequest, &ErrorResponse{Error: err.Error()})
			return
		}

		var from, to time.Time
		var err error
		if query.From != "" {
			from, err = time.Parse("2006-01-02", query.From)
			if err != nil {
				c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "неверный формат даты from, ожидается YYYY-MM-DD"})
				return
			}
		}
		if query.To != "" {
			to, err = time.Parse("2006-01-02", query.To)
			if err != nil {
				c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "неверный формат даты to, ожидается YYYY-MM-DD"})
				return
			}
		}

		items, err := svc.GetItems(c.Request.Context(), from, to, query.SortBy, query.SortOrder)
		if err != nil {
			log.Error("GetItemsPeriodSorted: ошибка вызова сервиса", "error", err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{Error: err.Error()})
			return
		}

		// преобразование в API-модели
		result := make([]*Item, len(items))
		for i, item := range items {
			result[i] = &Item{
				ID:       item.ID,
				Category: item.Category,
				Amount:   item.Amount,
				Date:     item.Date.Format("2006-01-02"),
			}
		}

		c.JSON(http.StatusOK, result)
	}
}

// CreateItem создаёт новую запись
func CreateItem(svc *service.Service, log logger.Logger) ginext.HandlerFunc {
	return func(c *ginext.Context) {

		req := CreateUpdateItemRequest{}
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Error("CreateItem: ошибка парсинга JSON", "error", err)
			c.JSON(http.StatusBadRequest, &ErrorResponse{Error: err.Error()})
			return
		}

		date, err := time.Parse("2006-01-02", req.Date)
		if err != nil {
			c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "неверный формат даты, ожидается YYYY-MM-DD"})
			return
		}

		domainItem := &domain.Item{
			Category: req.Category,
			Amount:   req.Amount,
			Date:     date,
		}

		created, err := svc.CreateItem(c.Request.Context(), domainItem)
		if err != nil {
			log.Error("CreateItem: ошибка вызова сервиса", "error", err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{Error: err.Error()})
			return
		}

		result := &Item{
			ID:       created.ID,
			Category: created.Category,
			Amount:   created.Amount,
			Date:     created.Date.Format("2006-01-02"),
		}

		c.JSON(http.StatusCreated, result)
	}
}

// UpdateItem обновляет существующую запись
func UpdateItem(svc *service.Service, log logger.Logger) ginext.HandlerFunc {
	return func(c *ginext.Context) {

		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "некорректный идентификатор"})
			return
		}

		req := CreateUpdateItemRequest{}
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Error("UpdateItem: ошибка парсинга JSON", "error", err)
			c.JSON(http.StatusBadRequest, &ErrorResponse{Error: err.Error()})
			return
		}

		date, err := time.Parse("2006-01-02", req.Date)
		if err != nil {
			c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "неверный формат даты, ожидается YYYY-MM-DD"})
			return
		}

		domainItem := &domain.Item{
			Category: req.Category,
			Amount:   req.Amount,
			Date:     date,
		}

		updated, err := svc.UpdateItem(c.Request.Context(), id, domainItem)
		if err != nil {
			log.Error("UpdateItem: ошибка вызова сервиса", "error", err)
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, &ErrorResponse{Error: "запись не найдена"})
				return
			}
			c.JSON(http.StatusInternalServerError, &ErrorResponse{Error: err.Error()})
			return
		}

		result := &Item{
			ID:       updated.ID,
			Category: updated.Category,
			Amount:   updated.Amount,
			Date:     updated.Date.Format("2006-01-02"),
		}

		c.JSON(http.StatusOK, result)
	}
}

// DelItem удаляет запись
func DelItem(svc *service.Service, log logger.Logger) ginext.HandlerFunc {
	return func(c *ginext.Context) {

		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "некорректный идентификатор"})
			return
		}

		err = svc.DeleteItem(c.Request.Context(), id)
		if err != nil {
			log.Error("DelItem: ошибка вызова сервиса", "error", err)
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, &ErrorResponse{Error: "запись не найдена"})
				return
			}
			c.JSON(http.StatusInternalServerError, &ErrorResponse{Error: err.Error()})
			return
		}

		c.Status(http.StatusNoContent)
	}
}
