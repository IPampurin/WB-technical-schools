package api

import (
	"context"
	"net/http"

	"github.com/IPampurin/UrlShortener/pkg/service"
	"github.com/gin-gonic/gin"
	"github.com/wb-go/wbf/logger"
)

// CreateShortLink обрабатывает POST /shorten
func CreateShortLink(svc service.ServiceMethods, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		var req CreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Ctx(c.Request.Context()).Error("неверный формат запроса", "error", err)
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "неверный формат запроса"})
			return
		}

		link, err := svc.CreateShortLink(c.Request.Context(), log, req.OriginalURL, req.CustomShort)
		if err != nil {
			log.Ctx(c.Request.Context()).Error("ошибка создания ссылки", "error", err)
			// здесь можно проверять конкретные ошибки, если нужно
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "внутренняя ошибка сервера"})
			return
		}

		c.JSON(http.StatusCreated, link)
	}
}

// Redirect обрабатывает GET /s/:short_url
func Redirect(svc service.ServiceMethods, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		shortURL := c.Param("short_url")

		link, err := svc.ShortLinkInfo(c.Request.Context(), log, shortURL)
		if err != nil {
			log.Ctx(c.Request.Context()).Error("ошибка получения ссылки", "error", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "внутренняя ошибка"})
			return
		}
		if link == nil {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "ссылка не найдена"})
			return
		}

		// асинхронно записываем аналитику
		go func(linkID int, userAgent, ip, referer string) {

			ctx := context.Background()
			if err := svc.RecordClick(ctx, log, linkID, userAgent, ip, referer); err != nil {
				log.Ctx(ctx).Error("ошибка записи аналитики", "error", err)
			}
			if err := svc.IncrementClicks(ctx, log, int64(linkID)); err != nil {
				log.Ctx(ctx).Error("ошибка увеличения счётчика", "error", err)
			}

		}(link.ID, c.GetHeader("User-Agent"), c.ClientIP(), c.GetHeader("Referer"))

		c.Redirect(http.StatusFound, link.OriginalURL)
	}
}

// GetAnalytics обрабатывает GET /analytics/:short_url
func GetAnalytics(svc service.ServiceMethods, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		shortURL := c.Param("short_url")

		analytics, err := svc.ShortLinkAnalytics(c.Request.Context(), log, shortURL)
		if err != nil {
			log.Ctx(c.Request.Context()).Error("ошибка получения аналитики", "error", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "внутренняя ошибка"})
			return
		}
		if analytics == nil {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "ссылка не найдена"})
			return
		}

		c.JSON(http.StatusOK, analytics)
	}
}

// GetLinks обрабатывает GET /links
func GetLinks(svc service.ServiceMethods, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		links, err := svc.LastLinks(c.Request.Context(), log)
		if err != nil {
			log.Ctx(c.Request.Context()).Error("ошибка получения списка ссылок", "error", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "внутренняя ошибка"})
			return
		}

		c.JSON(http.StatusOK, links)
	}
}

// SearchByOriginal обрабатывает GET /links/search/original?q=...
func SearchByOriginal(svc service.ServiceMethods, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		query := c.Query("q")
		if query == "" {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "параметр q обязателен"})
			return
		}

		links, err := svc.SearchByOriginalURL(c.Request.Context(), log, query)
		if err != nil {
			log.Ctx(c.Request.Context()).Error("ошибка поиска по оригинальному URL", "error", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "внутренняя ошибка"})
			return
		}

		c.JSON(http.StatusOK, links)
	}
}

// SearchByShort обрабатывает GET /links/search/short?q=...
func SearchByShort(svc service.ServiceMethods, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		query := c.Query("q")
		if query == "" {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "параметр q обязателен"})
			return
		}

		links, err := svc.SearchByShortURL(c.Request.Context(), log, query)
		if err != nil {
			log.Ctx(c.Request.Context()).Error("ошибка поиска по короткому URL", "error", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "внутренняя ошибка"})
			return
		}

		c.JSON(http.StatusOK, links)
	}
}
