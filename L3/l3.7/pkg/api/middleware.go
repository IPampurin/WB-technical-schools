package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/IPampurin/WarehouseControlAntiPattern/pkg/auth"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware проверяет наличие и валидность JWT в заголовке Authorization
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "отсутствует токен авторизации"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "неверный формат токена"})
			return
		}

		claims, err := auth.ParseToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "недействительный токен"})
			return
		}

		// сохраняем данные пользователя в контексте запроса
		ctx := context.WithValue(c.Request.Context(), auth.UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, auth.RoleKey, claims.Role)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
