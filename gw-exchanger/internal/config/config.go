package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// Config содержит всю конфигурацию приложения
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Logger   LoggerConfig
}

// ServerConfig содержит конфигурацию сервера
type ServerConfig struct {
	GRPCPort string
}

// DatabaseConfig содержит конфигурацию базы данных
type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// LoggerConfig содержит конфигурацию логгера
type LoggerConfig struct {
	Level string
}

// Load загружает конфигурацию из файла окружения
func Load(configPath string) (*Config, error) {
	// Загрузка переменных окружения из файла
	if configPath != "" {
		if err := godotenv.Load(configPath); err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
	}

	cfg := &Config{}

	// Загрузка конфигурации сервера
	cfg.Server.GRPCPort = getEnv("GRPC_PORT", DefaultGRPCPort)

	// Загрузка конфигурации базы данных
	cfg.Database.Host = getEnv("DB_HOST", DefaultDBHost)
	cfg.Database.Port = getEnvInt("DB_PORT", DefaultDBPort)
	cfg.Database.User = getEnv("DB_USER", DefaultDBUser)
	cfg.Database.Password = getEnv("DB_PASSWORD", DefaultDBPassword)
	cfg.Database.DBName = getEnv("DB_NAME", DefaultDBName)
	cfg.Database.SSLMode = getEnv("DB_SSLMODE", DefaultDBSSLMode)
	cfg.Database.MaxOpenConns = getEnvInt("DB_MAX_OPEN_CONNS", DefaultDBMaxOpenConns)
	cfg.Database.MaxIdleConns = getEnvInt("DB_MAX_IDLE_CONNS", DefaultDBMaxIdleConns)
	cfg.Database.ConnMaxLifetime = getEnvDuration("DB_CONN_MAX_LIFETIME", DefaultDBConnMaxLifetime)

	// Загрузка конфигурации логгера
	cfg.Logger.Level = getEnv("LOG_LEVEL", DefaultLogLevel)

	return cfg, nil
}

// getEnv получает переменную окружения или возвращает значение по умолчанию
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt получает целочисленную переменную окружения или возвращает значение по умолчанию
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvDuration получает переменную окружения типа duration или возвращает значение по умолчанию
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// Validate проверяет корректность конфигурации
func (c *Config) Validate() error {
	if c.Server.GRPCPort == "" {
		return fmt.Errorf("GRPC_PORT is required")
	}

	if c.Database.Host == "" {
		return fmt.Errorf("DB_HOST is required")
	}

	if c.Database.User == "" {
		return fmt.Errorf("DB_USER is required")
	}

	if c.Database.DBName == "" {
		return fmt.Errorf("DB_NAME is required")
	}

	// Проверка уровня логирования
	if _, err := logrus.ParseLevel(c.Logger.Level); err != nil {
		return fmt.Errorf("invalid log level: %s", c.Logger.Level)
	}

	return nil
}
