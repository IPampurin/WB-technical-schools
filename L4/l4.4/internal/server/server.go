package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/IPampurin/GC-MetricsServer/internal/configuration"
	"github.com/gin-gonic/gin"
)

const shutdownTimeout = 30 * time.Second

// Server - обёртка над gin.Engine и net/http.Server
type Server struct {
	cfg        *configuration.Config
	engine     *gin.Engine
	httpServer *http.Server
}

// New создаёт движок Gin, регистрирует маршруты
func New(cfg *configuration.Config) (*Server, error) {

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// раздача статики
	r.Static("/static", "./web")
	r.GET("/", func(c *gin.Context) {
		c.File("./web/index.html")
	})

	// остальные маршруты
	setupRoutes(r, cfg)

	s := &Server{
		cfg:    cfg,
		engine: r,
	}

	// создаём http.Server и сохраняем в структуру
	s.httpServer = &http.Server{
		Addr:    s.Addr(),
		Handler: s.engine,
	}

	return s, nil
}

// Addr - адрес прослушивания в формате host:port
func (s *Server) Addr() string {

	return fmt.Sprintf("%s:%s", s.cfg.Host, s.cfg.Port)
}

// Run запускает HTTP-сервер (использует s.httpServer)
func (s *Server) Run(ctx context.Context) error {

	log.Printf("Сервер запущен на http://%s", s.httpServer.Addr)
	log.Printf("Веб-интерфейс: http://%s/", s.httpServer.Addr)
	log.Printf("Prometheus метрики: http://%s/metrics", s.httpServer.Addr)
	log.Printf("Управление GOGC: GET/POST http://%s/gc_percent", s.httpServer.Addr)
	log.Printf("pprof профили: http://%s/debug/pprof/", s.httpServer.Addr)

	errCh := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return s.httpServer.Shutdown(shutCtx)
	case err := <-errCh:
		return err
	}
}
