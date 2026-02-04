package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// Config содержит всю конфигурацию приложения
type Config struct {
	Service    ServiceConfig
	MongoDB    MongoDBConfig
	Kafka      KafkaConfig
	Processing ProcessingConfig
	Logger     LoggerConfig
}

// ServiceConfig содержит конфигурацию сервиса
type ServiceConfig struct {
	Name string
}

// MongoDBConfig содержит конфигурацию MongoDB
type MongoDBConfig struct {
	URI         string
	Database    string
	Collection  string
	Timeout     time.Duration
	MaxPoolSize uint64
	MinPoolSize uint64
}

// KafkaConfig содержит конфигурацию Kafka
type KafkaConfig struct {
	Brokers   []string
	Topic     string
	GroupID   string
	Partition int
	MinBytes  int
	MaxBytes  int
	MaxWait   time.Duration
}

// ProcessingConfig содержит конфигурацию обработки
type ProcessingConfig struct {
	BatchSize          int
	Workers            int
	FlushInterval      time.Duration
	MaxProcessingTime  time.Duration
	RetryAttempts      int
	RetryDelay         time.Duration
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

	// Service
	cfg.Service.Name = getEnv("SERVICE_NAME", DefaultServiceName)

	// MongoDB
	cfg.MongoDB.URI = getEnv("MONGO_URI", DefaultMongoURI)
	cfg.MongoDB.Database = getEnv("MONGO_DATABASE", DefaultMongoDatabase)
	cfg.MongoDB.Collection = getEnv("MONGO_COLLECTION", DefaultMongoCollection)
	cfg.MongoDB.Timeout = getEnvDuration("MONGO_TIMEOUT", DefaultMongoTimeout)
	cfg.MongoDB.MaxPoolSize = uint64(getEnvInt("MONGO_MAX_POOL_SIZE", DefaultMongoMaxPoolSize))
	cfg.MongoDB.MinPoolSize = uint64(getEnvInt("MONGO_MIN_POOL_SIZE", DefaultMongoMinPoolSize))

	// Kafka
	brokers := getEnv("KAFKA_BROKERS", DefaultKafkaBrokers)
	cfg.Kafka.Brokers = strings.Split(brokers, ",")
	cfg.Kafka.Topic = getEnv("KAFKA_TOPIC", DefaultKafkaTopic)
	cfg.Kafka.GroupID = getEnv("KAFKA_GROUP_ID", DefaultKafkaGroupID)
	cfg.Kafka.Partition = getEnvInt("KAFKA_PARTITION", DefaultKafkaPartition)
	cfg.Kafka.MinBytes = getEnvInt("KAFKA_MIN_BYTES", DefaultKafkaMinBytes)
	cfg.Kafka.MaxBytes = getEnvInt("KAFKA_MAX_BYTES", DefaultKafkaMaxBytes)
	cfg.Kafka.MaxWait = getEnvDuration("KAFKA_MAX_WAIT", DefaultKafkaMaxWait)

	// Processing
	cfg.Processing.BatchSize = getEnvInt("BATCH_SIZE", DefaultBatchSize)
	cfg.Processing.Workers = getEnvInt("WORKERS", DefaultWorkers)
	cfg.Processing.FlushInterval = getEnvDuration("FLUSH_INTERVAL", DefaultFlushInterval)
	cfg.Processing.MaxProcessingTime = getEnvDuration("MAX_PROCESSING_TIME", DefaultMaxProcessingTime)
	cfg.Processing.RetryAttempts = getEnvInt("RETRY_ATTEMPTS", DefaultRetryAttempts)
	cfg.Processing.RetryDelay = getEnvDuration("RETRY_DELAY", DefaultRetryDelay)

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
	if c.MongoDB.URI == "" {
		return fmt.Errorf("MONGO_URI is required")
	}

	if c.MongoDB.Database == "" {
		return fmt.Errorf("MONGO_DATABASE is required")
	}

	if len(c.Kafka.Brokers) == 0 {
		return fmt.Errorf("KAFKA_BROKERS is required")
	}

	if c.Kafka.Topic == "" {
		return fmt.Errorf("KAFKA_TOPIC is required")
	}

	if c.Processing.BatchSize <= 0 {
		return fmt.Errorf("BATCH_SIZE must be positive")
	}

	if c.Processing.Workers <= 0 {
		return fmt.Errorf("WORKERS must be positive")
	}

	if _, err := logrus.ParseLevel(c.Logger.Level); err != nil {
		return fmt.Errorf("invalid log level: %s", c.Logger.Level)
	}

	return nil
}
