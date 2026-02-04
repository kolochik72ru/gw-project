package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Logger middleware для логирования HTTP запросов
func Logger(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Начало запроса
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// Обработка запроса
		c.Next()

		// Конец запроса
		duration := time.Since(start)
		statusCode := c.Writer.Status()

		// Логирование
		entry := logger.WithFields(logrus.Fields{
			"method":   method,
			"path":     path,
			"status":   statusCode,
			"duration": duration.String(),
			"client_ip": c.ClientIP(),
		})

		if len(c.Errors) > 0 {
			entry.Error(c.Errors.String())
		} else {
			if statusCode >= 500 {
				entry.Error("Internal server error")
			} else if statusCode >= 400 {
				entry.Warn("Client error")
			} else {
				entry.Info("Request completed")
			}
		}
	}
}
