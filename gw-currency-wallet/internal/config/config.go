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
	Server    ServerConfig
	Database  DatabaseConfig
	JWT       JWTConfig
	Exchanger ExchangerConfig
	Cache     CacheConfig
	Kafka     KafkaConfig
	Logger    LoggerConfig
}

// ServerConfig содержит конфигурацию сервера
type ServerConfig struct {
	HTTPPort string
	GinMode  string
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

// JWTConfig содержит конфигурацию JWT
type JWTConfig struct {
	Secret     string
	Expiration time.Duration
}

// ExchangerConfig содержит конфигурацию gRPC клиента для exchanger
type ExchangerConfig struct {
	Host    string
	Port    string
	Timeout time.Duration
}

// CacheConfig содержит конфигурацию кеша
type CacheConfig struct {
	RatesTTL time.Duration
}

// KafkaConfig содержит конфигурацию Kafka
type KafkaConfig struct {
	Brokers           []string
	Topic             string
	TransferThreshold float64
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

	// Server
	cfg.Server.HTTPPort = getEnv("HTTP_PORT", DefaultHTTPPort)
	cfg.Server.GinMode = getEnv("GIN_MODE", DefaultGinMode)

	// Database
	cfg.Database.Host = getEnv("DB_HOST", DefaultDBHost)
	cfg.Database.Port = getEnvInt("DB_PORT", DefaultDBPort)
	cfg.Database.User = getEnv("DB_USER", DefaultDBUser)
	cfg.Database.Password = getEnv("DB_PASSWORD", DefaultDBPassword)
	cfg.Database.DBName = getEnv("DB_NAME", DefaultDBName)
	cfg.Database.SSLMode = getEnv("DB_SSLMODE", DefaultDBSSLMode)
	cfg.Database.MaxOpenConns = getEnvInt("DB_MAX_OPEN_CONNS", DefaultDBMaxOpenConns)
	cfg.Database.MaxIdleConns = getEnvInt("DB_MAX_IDLE_CONNS", DefaultDBMaxIdleConns)
	cfg.Database.ConnMaxLifetime = getEnvDuration("DB_CONN_MAX_LIFETIME", DefaultDBConnMaxLifetime)

	// JWT
	cfg.JWT.Secret = getEnv("JWT_SECRET", DefaultJWTSecret)
	cfg.JWT.Expiration = getEnvDuration("JWT_EXPIRATION", DefaultJWTExpiration)

	// Exchanger gRPC
	cfg.Exchanger.Host = getEnv("EXCHANGER_GRPC_HOST", DefaultExchangerHost)
	cfg.Exchanger.Port = getEnv("EXCHANGER_GRPC_PORT", DefaultExchangerPort)
	cfg.Exchanger.Timeout = getEnvDuration("EXCHANGER_GRPC_TIMEOUT", DefaultExchangerTimeout)

	// Cache
	cfg.Cache.RatesTTL = getEnvDuration("CACHE_RATES_TTL", DefaultCacheRatesTTL)

	// Kafka
	brokers := getEnv("KAFKA_BROKERS", DefaultKafkaBrokers)
	cfg.Kafka.Brokers = []string{brokers} // В продакшене можно разбить по запятой
	cfg.Kafka.Topic = getEnv("KAFKA_TOPIC", DefaultKafkaTopic)
	cfg.Kafka.TransferThreshold = getEnvFloat("KAFKA_TRANSFER_THRESHOLD", DefaultKafkaTransferThreshold)

	// Logger
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

// getEnvInt получает целочисленную переменную окружения
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvFloat получает переменную окружения типа float64
func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

// getEnvDuration получает переменную окружения типа duration
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
	if c.Server.HTTPPort == "" {
		return fmt.Errorf("HTTP_PORT is required")
	}

	if c.Database.Host == "" {
		return fmt.Errorf("DB_HOST is required")
	}

	if c.JWT.Secret == "" || c.JWT.Secret == "your-super-secret-jwt-key-change-this-in-production" {
		return fmt.Errorf("JWT_SECRET must be set to a secure value")
	}

	if _, err := logrus.ParseLevel(c.Logger.Level); err != nil {
		return fmt.Errorf("invalid log level: %s", c.Logger.Level)
	}

	return nil
}
