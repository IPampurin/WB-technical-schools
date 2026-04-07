package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/IPampurin/pprofSimpleService/internal/configuration"
	"github.com/IPampurin/pprofSimpleService/internal/interfaces"
)

// Server инкапсулирует HTTP-сервер и его зависимости
type Server struct {
	httpSrv *http.Server
	engine  *gin.Engine
	cfg     *configuration.Config
}

// NewServer создаёт новый экземпляр сервера
func NewServer(cfg *configuration.Config, svc interfaces.ServiceMethods) *Server {

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// статические файлы и главная страница
	r.Static("/static", "./web")
	r.GET("/", func(c *gin.Context) {
		c.File("./web/index.html")
	})

	// создаём обработчик API
	h := &handler{svc: svc}

	// регистрируем эндпоинты
	r.GET("/sum", h.Sum)
	r.GET("/fib", h.Fib)
	r.POST("/allocate", h.Allocate)

	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	httpSrv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	return &Server{
		httpSrv: httpSrv,
		engine:  r,
		cfg:     cfg,
	}
}

// Run запускает сервер и ожидает отмены контекста для graceful shutdown
func (s *Server) Run(ctx context.Context) error {

	errCh := make(chan error, 1)
	go func() {
		if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
		defer cancel()
		return s.httpSrv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
