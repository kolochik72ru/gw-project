package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

// Claims структура JWT claims
type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// JWTMiddleware middleware для проверки JWT токенов
type JWTMiddleware struct {
	secret []byte
	logger *logrus.Logger
}

// NewJWTMiddleware создает новый JWT middleware
func NewJWTMiddleware(secret string, logger *logrus.Logger) *JWTMiddleware {
	return &JWTMiddleware{
		secret: []byte(secret),
		logger: logger,
	}
}

// Auth middleware для аутентификации
func (m *JWTMiddleware) Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем токен из заголовка Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		// Проверяем формат "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Парсим и валидируем токен
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			// Проверяем алгоритм подписи
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return m.secret, nil
		})

		if err != nil {
			m.logger.Warnf("Invalid token: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Извлекаем claims
		if claims, ok := token.Claims.(*Claims); ok && token.Valid {
			// Сохраняем данные пользователя в контекст
			c.Set("user_id", claims.UserID)
			c.Set("username", claims.Username)
			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}
	}
}

// GenerateToken генерирует JWT токен для пользователя
func (m *JWTMiddleware) GenerateToken(userID int64, username string, expiration time.Duration) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(m.secret)
	if err != nil {
		m.logger.Errorf("Failed to sign token: %v", err)
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return tokenString, nil
}

// GetUserID извлекает user_id из контекста
func GetUserID(c *gin.Context) (int64, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, fmt.Errorf("user_id not found in context")
	}

	id, ok := userID.(int64)
	if !ok {
		return 0, fmt.Errorf("invalid user_id type")
	}

	return id, nil
}

// GetUsername извлекает username из контекста
func GetUsername(c *gin.Context) (string, error) {
	username, exists := c.Get("username")
	if !exists {
		return "", fmt.Errorf("username not found in context")
	}

	name, ok := username.(string)
	if !ok {
		return "", fmt.Errorf("invalid username type")
	}

	return name, nil
}
