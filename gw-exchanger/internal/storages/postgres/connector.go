package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// Config содержит конфигурацию для подключения к PostgreSQL
type Config struct {
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

// PostgresStorage реализует интерфейс Storage для PostgreSQL
type PostgresStorage struct {
	db     *sql.DB
	logger *logrus.Logger
}

// New создает новое подключение к PostgreSQL
func New(cfg *Config, logger *logrus.Logger) (*PostgresStorage, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Настройка пула соединений
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Проверка подключения
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Successfully connected to PostgreSQL")

	storage := &PostgresStorage{
		db:     db,
		logger: logger,
	}

	// Инициализация схемы БД
	if err := storage.initSchema(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return storage, nil
}

// initSchema создает необходимые таблицы, если они не существуют
func (s *PostgresStorage) initSchema(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS currencies (
		id SERIAL PRIMARY KEY,
		code VARCHAR(3) UNIQUE NOT NULL,
		name VARCHAR(100) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS exchange_rates (
		id SERIAL PRIMARY KEY,
		from_currency VARCHAR(3) NOT NULL,
		to_currency VARCHAR(3) NOT NULL,
		rate NUMERIC(20, 8) NOT NULL,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(from_currency, to_currency)
	);

	CREATE INDEX IF NOT EXISTS idx_exchange_rates_currencies 
		ON exchange_rates(from_currency, to_currency);
	`

	_, err := s.db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	s.logger.Info("Database schema initialized")

	// Добавляем начальные данные, если таблица пустая
	return s.seedInitialData(ctx)
}

// seedInitialData добавляет начальные данные о валютах и курсах
func (s *PostgresStorage) seedInitialData(ctx context.Context) error {
	// Проверяем, есть ли уже данные
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM currencies").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		s.logger.Info("Database already contains data, skipping seed")
		return nil
	}

	// Добавляем валюты
	currencies := []struct {
		code string
		name string
	}{
		{"USD", "US Dollar"},
		{"EUR", "Euro"},
		{"RUB", "Russian Ruble"},
	}

	for _, curr := range currencies {
		_, err := s.db.ExecContext(ctx,
			"INSERT INTO currencies (code, name) VALUES ($1, $2) ON CONFLICT (code) DO NOTHING",
			curr.code, curr.name,
		)
		if err != nil {
			return fmt.Errorf("failed to insert currency %s: %w", curr.code, err)
		}
	}

	// Добавляем начальные курсы обмена
	rates := []struct {
		from string
		to   string
		rate float64
	}{
		{"USD", "EUR", 0.92},
		{"USD", "RUB", 92.50},
		{"EUR", "USD", 1.09},
		{"EUR", "RUB", 100.54},
		{"RUB", "USD", 0.0108},
		{"RUB", "EUR", 0.0099},
	}

	for _, rate := range rates {
		_, err := s.db.ExecContext(ctx,
			"INSERT INTO exchange_rates (from_currency, to_currency, rate) VALUES ($1, $2, $3) ON CONFLICT (from_currency, to_currency) DO NOTHING",
			rate.from, rate.to, rate.rate,
		)
		if err != nil {
			return fmt.Errorf("failed to insert rate %s->%s: %w", rate.from, rate.to, err)
		}
	}

	s.logger.Info("Initial data seeded successfully")
	return nil
}

// Close закрывает соединение с базой данных
func (s *PostgresStorage) Close() error {
	if s.db != nil {
		s.logger.Info("Closing database connection")
		return s.db.Close()
	}
	return nil
}

// Ping проверяет соединение с базой данных
func (s *PostgresStorage) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}
