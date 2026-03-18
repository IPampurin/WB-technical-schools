package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// тип для ключей контекста, чтобы избежать коллизий
type contextKey string

const (
	UserIDKey contextKey = "userID" // ключ для хранения id пользователя в контексте
	RoleKey   contextKey = "role"   // ключ для хранения роли пользователя в контексте
)

// secretKey — ключ для подписи JWT
var secretKey []byte

// Claims — структура утверждений JWT, содержит ID пользователя и его роль
type Claims struct {
	UserID int    `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// Init устанавливает секретный ключ для подписи JWT
func Init(key string) {

	secretKey = []byte(key)
}

// GenerateToken создаёт новый JWT для пользователя с указанным ID и ролью
func GenerateToken(userID int, role string) (string, error) {

	if len(secretKey) == 0 {
		return "", errors.New("секретный ключ не инициализирован")
	}

	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // срок действия 24 часа
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(secretKey)
}

// ParseToken проверяет токен и возвращает утверждения claims
func ParseToken(tokenString string) (*Claims, error) {

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// проверяем метод подписи
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("неверный метод подписи")
		}
		return secretKey, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("недействительный токен")
}
