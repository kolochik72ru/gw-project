package config

import "time"

// Server defaults
const (
	DefaultHTTPPort = "8080"
	DefaultGinMode  = "release"
	DefaultLogLevel = "info"
)

// Database defaults
const (
	DefaultDBHost            = "localhost"
	DefaultDBPort            = 5432
	DefaultDBUser            = "wallet_user"
	DefaultDBPassword        = "wallet_password"
	DefaultDBName            = "wallet_db"
	DefaultDBSSLMode         = "disable"
	DefaultDBMaxOpenConns    = 25
	DefaultDBMaxIdleConns    = 5
	DefaultDBConnMaxLifetime = 5 * time.Minute
)

// JWT defaults
const (
	DefaultJWTSecret     = "change-me-in-production"
	DefaultJWTExpiration = 24 * time.Hour
)

// Exchanger gRPC defaults
const (
	DefaultExchangerHost    = "localhost"
	DefaultExchangerPort    = "50051"
	DefaultExchangerTimeout = 5 * time.Second
)

// Cache defaults
const (
	DefaultCacheRatesTTL = 5 * time.Minute
)

// Kafka defaults
const (
	DefaultKafkaBrokers           = "localhost:9092"
	DefaultKafkaTopic             = "large-transfers"
	DefaultKafkaTransferThreshold = 30000.0
)
