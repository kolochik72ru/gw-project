package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

// New создает новый настроенный логгер
func New(level string) *logrus.Logger {
	logger := logrus.New()

	// Установка формата вывода
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})

	// Установка уровня логирования
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)

	// Вывод в stdout
	logger.SetOutput(os.Stdout)

	return logger
}

// WithFields добавляет дополнительные поля к логгеру
func WithFields(logger *logrus.Logger, fields map[string]interface{}) *logrus.Entry {
	return logger.WithFields(fields)
}
