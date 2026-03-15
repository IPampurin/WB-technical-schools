package api

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/IPampurin/SalesTracker/pkg/service"
	"github.com/wb-go/wbf/ginext"
	"github.com/wb-go/wbf/logger"
)

// GetAnalytic возвращает общие метрики за период
func GetAnalytic(svc *service.Service, log logger.Logger) ginext.HandlerFunc {
	return func(c *ginext.Context) {

		query := AnalyticsQuery{}
		if err := c.ShouldBindQuery(&query); err != nil {
			log.Error("GetAnalytic: ошибка парсинга query", "error", err)
			c.JSON(http.StatusBadRequest, &ErrorResponse{Error: err.Error()})
			return
		}

		from, err := time.Parse("2006-01-02", query.From)
		if err != nil {
			c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "неверный формат даты from"})
			return
		}

		to, err := time.Parse("2006-01-02", query.To)
		if err != nil {
			c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "неверный формат даты to"})
			return
		}

		sum, avg, count, median, percentile90, err := svc.GetAnalytics(c.Request.Context(), from, to)
		if err != nil {
			log.Error("GetAnalytic: ошибка вызова сервиса", "error", err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{Error: err.Error()})
			return
		}

		resp := &AnalyticsResponse{
			Sum:          sum,
			Avg:          avg,
			Count:        count,
			Median:       median,
			Percentile90: percentile90,
		}

		c.JSON(http.StatusOK, resp)
	}
}

// GetAnalyticGroupCategory возвращает агрегированные данные по категориям
func GetAnalyticGroupCategory(svc *service.Service, log logger.Logger) ginext.HandlerFunc {
	return func(c *ginext.Context) {

		query := AnalyticsQuery{}
		if err := c.ShouldBindQuery(&query); err != nil {
			log.Error("GetAnalyticGroupCategory: ошибка парсинга query", "error", err)
			c.JSON(http.StatusBadRequest, &ErrorResponse{Error: err.Error()})
			return
		}

		from, err := time.Parse("2006-01-02", query.From)
		if err != nil {
			c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "неверный формат даты from"})
			return
		}

		to, err := time.Parse("2006-01-02", query.To)
		if err != nil {
			c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "неверный формат даты to"})
			return
		}

		aggs, err := svc.GetAnalyticsByCategory(c.Request.Context(), from, to)
		if err != nil {
			log.Error("GetAnalyticGroupCategory: ошибка вызова сервиса", "error", err)
			c.JSON(http.StatusInternalServerError, &ErrorResponse{Error: err.Error()})
			return
		}

		result := make([]*CategoryGroup, len(aggs))
		for i, agg := range aggs {
			result[i] = &CategoryGroup{
				Category: agg.Category,
				Sum:      agg.Sum,
				Count:    agg.Count,
			}
		}

		c.JSON(http.StatusOK, result)
	}
}

// ExportCSVFile экспортирует записи за период в CSV
func ExportCSVFile(svc *service.Service, log logger.Logger) ginext.HandlerFunc {
	return func(c *ginext.Context) {

		query := ExportCSVQuery{}
		if err := c.ShouldBindQuery(&query); err != nil {
			log.Error("ExportCSVFile: ошибка парсинга query", "error", err)
			c.JSON(http.StatusBadRequest, &ErrorResponse{Error: err.Error()})
			return
		}

		from, err := time.Parse("2006-01-02", query.From)
		if err != nil {
			c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "неверный формат даты from"})
			return
		}

		to, err := time.Parse("2006-01-02", query.To)
		if err != nil {
			c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "неверный формат даты to"})
			return
		}

		// вызываем метод сервиса для генерации CSV
		csvData, err := svc.ExportCSV(c.Request.Context(), from, to)
		if err != nil {
			log.Error("ExportCSVFile: ошибка вызова сервиса", "error", err)
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, &ErrorResponse{Error: "нет данных за указанный период"})
				return
			}
			c.JSON(http.StatusInternalServerError, &ErrorResponse{Error: err.Error()})
			return
		}

		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", "attachment; filename=sales_"+query.From+"_"+query.To+".csv")
		c.Data(http.StatusOK, "text/csv; charset=utf-8", csvData)
	}
}
