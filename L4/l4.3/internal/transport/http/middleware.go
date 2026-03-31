// Middleware для HTTP (логирование запросов, request id и т.д.)
package http

import (
	"time"

	"github.com/IPampurin/EventCalendar/internal/service"
	"github.com/gin-gonic/gin"
)

func LoggingMiddleware(logger service.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		start := time.Now()
		c.Next()

		logger.Info("HTTP request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration", time.Since(start),
		)
	}
}
