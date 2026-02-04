package config

import "time"

// Service defaults
const (
	DefaultServiceName = "gw-notification"
	DefaultLogLevel    = "info"
)

// MongoDB defaults
const (
	DefaultMongoURI         = "mongodb://localhost:27017"
	DefaultMongoDatabase    = "notification_db"
	DefaultMongoCollection  = "large_transfers"
	DefaultMongoTimeout     = 10 * time.Second
	DefaultMongoMaxPoolSize = 100
	DefaultMongoMinPoolSize = 10
)

// Kafka defaults
const (
	DefaultKafkaBrokers   = "localhost:9092"
	DefaultKafkaTopic     = "large-transfers"
	DefaultKafkaGroupID   = "notification-service-group"
	DefaultKafkaPartition = 0
	DefaultKafkaMinBytes  = 1
	DefaultKafkaMaxBytes  = 10485760 // 10MB
	DefaultKafkaMaxWait   = 500 * time.Millisecond
)

// Processing defaults
const (
	DefaultBatchSize          = 100
	DefaultWorkers            = 10
	DefaultFlushInterval      = 5 * time.Second
	DefaultMaxProcessingTime  = 30 * time.Second
	DefaultRetryAttempts      = 3
	DefaultRetryDelay         = 1 * time.Second
)
