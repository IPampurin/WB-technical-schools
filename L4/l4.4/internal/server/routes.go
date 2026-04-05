package server

import (
	"net/http"

	_ "net/http/pprof"

	"github.com/IPampurin/GC-MetricsServer/internal/collector"
	"github.com/IPampurin/GC-MetricsServer/internal/configuration"
	"github.com/IPampurin/GC-MetricsServer/internal/handlers"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// setupRoutes настраивает все маршруты, кроме статики и главной страницы
func setupRoutes(r *gin.Engine, cfg *configuration.Config) {

	// Prometheus метрики
	reg := prometheus.NewRegistry()
	reg.MustRegister(collector.NewMemStatsCollector())
	r.GET("/metrics", gin.WrapH(promhttp.HandlerFor(reg, promhttp.HandlerOpts{})))

	// API для получения метрик в JSON
	r.GET("/api/stats", handlers.HandleAPIMetrics)

	// управление GOGC
	r.GET("/gc_percent", handlers.HandleGCPercent)
	r.POST("/gc_percent", handlers.HandleGCPercent)

	// конфигурация для фронтенда (интервал обновления)
	r.GET("/api/config", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"metrics_timeout": cfg.MetricsTimeout.Seconds()})
	})

	// pprof профилирование
	r.Any("/debug/pprof/*any", gin.WrapH(http.DefaultServeMux))
}
