package handlers

import (
	"net/http"
	"runtime/debug"
	"strconv"

	"github.com/IPampurin/GC-MetricsServer/internal/collector"

	"github.com/gin-gonic/gin"
)

// HandleAPIMetrics возвращает текущие метрики памяти и GC в формате JSON
func HandleAPIMetrics(c *gin.Context) {
	c.JSON(http.StatusOK, collector.GetCurrentMetrics())
}

// HandleGCPercent обрабатывает GET и POST запросы для чтения/изменения GOGC
func HandleGCPercent(c *gin.Context) {

	switch c.Request.Method {
	case http.MethodGet:
		// Получаем текущий процент без изменения
		percent := debug.SetGCPercent(-1)
		debug.SetGCPercent(percent) // восстанавливаем
		c.JSON(http.StatusOK, gin.H{"gc_percent": percent})

	case http.MethodPost:
		percentStr := c.Query("percent")
		if percentStr == "" {
			c.String(http.StatusBadRequest, "Не указан параметр 'percent'")
			return
		}
		percent, err := strconv.Atoi(percentStr)
		if err != nil {
			c.String(http.StatusBadRequest, "Некорректное значение percent, ожидается целое число")
			return
		}
		old := debug.SetGCPercent(percent)
		c.JSON(http.StatusOK, gin.H{
			"old_percent": old,
			"new_percent": percent,
		})

	default:
		c.String(http.StatusMethodNotAllowed, "Метод не разрешён")
	}
}
