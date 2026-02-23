package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/IPampurin/DelayedNotifier/pkg/cache"
	"github.com/IPampurin/DelayedNotifier/pkg/db"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wb-go/wbf/logger"
)

// CreateNotificationHandler обрабатывает POST /notify
func CreateNotificationHandler(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		var req struct {
			UserID  int       `json:"user_id" binding:"required"`
			Channel []string  `json:"channel" binding:"required"`
			Content string    `json:"content" binding:"required"`
			SendFor time.Time `json:"send_for" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Ctx(c.Request.Context()).Error("неверный формат запроса", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "неверный формат запроса"})
			return
		}

		if req.SendFor.Before(time.Now()) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "время отправки должно быть в будущем"})
			return
		}

		// генерируем новый UUID
		uid := uuid.New()
		key := uid.String()

		// проверяем, нет ли уже в кэше (на всякий случай)
		if cachedJSON, err := cache.GetClientRedis().Get(c.Request.Context(), key); err == nil && cachedJSON != "" {
			c.JSON(http.StatusConflict, gin.H{"error": "уведомление с таким UID уже существует"})
			return
		}

		// создаём структуру
		now := time.Now()
		n := &db.Notification{
			UID:        uid,
			UserID:     req.UserID,
			Channel:    req.Channel,
			Content:    req.Content,
			Status:     "scheduled",
			SendFor:    req.SendFor,
			SendAt:     sql.NullTime{Valid: false},
			RetryCount: 0,
			LastError:  "",
			CreatedAt:  now,
		}

		// сохраняем в БД
		if err := db.GetClientDB().CreateNotification(c.Request.Context(), n); err != nil {
			log.Ctx(c.Request.Context()).Error("ошибка базы данных", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка базы данных"})
			return
		}

		// сериализуем в JSON и сохраняем в Redis
		if jsonData, err := json.Marshal(n); err == nil {
			_ = cache.GetClientRedis().SetWithExpiration(c.Request.Context(), key, string(jsonData), 10*time.Minute)
		} else {
			log.Ctx(c.Request.Context()).Error("ошибка маршаллинга уведомления", "error", err)
		}

		c.JSON(http.StatusCreated, n)
	}
}

// GetNotificationHandler обрабатывает GET /notify/:id
func GetNotificationHandler(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		idStr := c.Param("id")
		uid, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный UUID"})
			return
		}
		key := uid.String()

		// сначала пробуем получить из кэша
		if cachedJSON, err := cache.GetClientRedis().Get(c.Request.Context(), key); err == nil && cachedJSON != "" {
			var n db.Notification
			if err := json.Unmarshal([]byte(cachedJSON), &n); err == nil {
				c.JSON(http.StatusOK, n)
				return
			} else {
				log.Ctx(c.Request.Context()).Error("ошибка анмаршаллинга уведомления из кэша", "error", err)
				_ = cache.GetClientRedis().Del(c.Request.Context(), key) // удаляем битое значение
			}
		}

		// ищем в БД
		n, err := db.GetClientDB().GetNotification(c.Request.Context(), uid)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{"error": "уведомление не найдено"})
				return
			}
			log.Ctx(c.Request.Context()).Error("ошибка базы данных", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка базы данных"})
			return
		}

		// кладём в кэш для будущих запросов
		if jsonData, err := json.Marshal(n); err == nil {
			_ = cache.GetClientRedis().SetWithExpiration(c.Request.Context(), key, string(jsonData), 10*time.Minute)
		} else {
			log.Ctx(c.Request.Context()).Error("ошибка маршаллинга уведомления", "error", err)
		}

		c.JSON(http.StatusOK, n)
	}
}

// DeleteNotificationHandler обрабатывает DELETE /notify/:id
func DeleteNotificationHandler(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		idStr := c.Param("id")
		uid, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный UUID"})
			return
		}

		key := uid.String()

		// удаляем из кэша если есть
		_ = cache.GetClientRedis().Del(c.Request.Context(), key)

		// пытаемся отменить в БД
		err = db.GetClientDB().CancelNotification(c.Request.Context(), uid)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{"error": "уведомление не найдено или уже отменено"})
				return
			}
			log.Ctx(c.Request.Context()).Error("ошибка базы данных", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка базы данных"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "отменено"})
	}
}
