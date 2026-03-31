// HTTP-сервер на Gin: маршруты, таймауты и graceful shutdown по конфигу
package http

import (
	"context"
	"errors"
	"fmt"
	nethttp "net/http"
	"time"

	"github.com/IPampurin/EventCalendar/internal/configuration"
	"github.com/IPampurin/EventCalendar/internal/service"
	"github.com/gin-gonic/gin"
)

// Server - обёртка над gin.Engine и net/http.Server
type Server struct {
	cfg     *configuration.HTTPConfig
	engine  *gin.Engine
	handler *Handler
}

// NewServer - создаёт движок Gin, регистрирует маршруты календаря
func NewServer(cfg *configuration.HTTPConfig, svc *service.CalendarService, logger service.Logger) *Server {

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(LoggingMiddleware(logger))

	// раздача статики
	r.Static("/static", "./web")
	r.GET("/", func(c *gin.Context) {
		c.File("./web/index.html")
	})

	h := NewHandler(svc, logger)
	s := &Server{
		cfg:     cfg,
		engine:  r,
		handler: h,
	}
	s.registerRoutes()

	return s
}

// Addr - адрес прослушивания в формате host:port
func (s *Server) Addr() string {

	return fmt.Sprintf("%s:%s", s.cfg.Host, s.cfg.Port)
}

// Run - ListenAndServe до отмены ctx, затем Shutdown с таймаутом из конфига
func (s *Server) Run(ctx context.Context) error {

	shutdownTimeout := s.cfg.ShutdownTimeout
	if shutdownTimeout <= 0 {
		shutdownTimeout = 30 * time.Second
	}

	srv := &nethttp.Server{
		Addr:         s.Addr(),
		Handler:      s.engine,
		ReadTimeout:  s.cfg.ReadTimeout,
		WriteTimeout: s.cfg.WriteTimeout,
		IdleTimeout:  s.cfg.IdleTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return srv.Shutdown(shutCtx)

	case err := <-errCh:
		return err
	}
}
