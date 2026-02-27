package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/IPampurin/UrlShortener/pkg/api"
	"github.com/IPampurin/UrlShortener/pkg/configuration"
	"github.com/IPampurin/UrlShortener/pkg/service"
	"github.com/gin-gonic/gin"
	"github.com/wb-go/wbf/ginext"
	"github.com/wb-go/wbf/logger"
)

func Run(ctx context.Context, cfgServer *configuration.ConfServer, service *service.Service, log logger.Logger) error {

	// создаём движок Gin через обёртку ginext
	engine := ginext.New(cfgServer.GinMode)

	// добавляем middleware (логгер и восстановление)
	engine.Use(ginext.Logger(), ginext.Recovery())

	// добавляем свой middleware для структурного логирования запросов
	engine.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		// используем переданный логгер для записи информации о запросе
		log.LogRequest(c.Request.Context(), c.Request.Method, c.Request.URL.Path, c.Writer.Status(), duration)
	})

	// регистрируем эндпоинты
	engine.POST("/shorten", api.CreateShortLink(service, log))               // создание новой сокращённой ссылки
	engine.GET("/s/:short_url", api.Redirect(service, log))                  // переход по короткой ссылке
	engine.GET("/analytics/:short_url", api.GetAnalytics(service, log))      // получение аналитики (число и время переходов, User-Agent)
	engine.GET("/links", api.GetLinks(service, log))                         // список последних ссылок для UI
	engine.GET("/links/search/original", api.SearchByOriginal(service, log)) // поиск по OriginalURL
	engine.GET("/links/search/short", api.SearchByShort(service, log))       // поиск по ShortURL

	// раздаём статические файлы из папки ./web
	engine.Static("/static", "./web")

	// для корневого пути отдаём index.html
	engine.GET("/", func(c *ginext.Context) {
		c.File("./web/index.html")
	})

	// формируем адрес запуска
	addr := fmt.Sprintf("%s:%d", cfgServer.HostName, cfgServer.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: engine, // engine реализует http.Handler
	}

	// канал для ошибок от сервера
	errCh := make(chan error, 1)

	// запускаем сервер в горутине
	go func() {
		log.Info("запуск HTTP-сервера", "address", addr)
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// ожидаем либо сигнала от контекста, либо ошибки запуска
	select {
	case <-ctx.Done():
		log.Info("получен сигнал завершения, останавливаем сервер...")
		// даём время на завершение текущих запросов (например, 5 секунд)
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error("ошибка при graceful shutdown", "error", err)
			return err
		}
		log.Info("сервер корректно остановлен")
		return nil

	case err := <-errCh:
		log.Error("сервер завершился с ошибкой", "error", err)
		return err
	}
}
