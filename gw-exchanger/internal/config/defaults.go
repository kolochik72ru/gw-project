package config

import "time"

// Значения по умолчанию для конфигурации сервера
const (
	DefaultGRPCPort = "50051"
	DefaultLogLevel = "info"
)

// Значения по умолчанию для конфигурации базы данных
const (
	DefaultDBHost            = "localhost"
	DefaultDBPort            = 5432
	DefaultDBUser            = "exchanger_user"
	DefaultDBPassword        = "exchanger_password"
	DefaultDBName            = "exchanger_db"
	DefaultDBSSLMode         = "disable"
	DefaultDBMaxOpenConns    = 25
	DefaultDBMaxIdleConns    = 5
	DefaultDBConnMaxLifetime = 5 * time.Minute
)
