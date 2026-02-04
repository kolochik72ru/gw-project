package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Config содержит конфигурацию для подключения к MongoDB
type Config struct {
	URI            string
	Database       string
	Collection     string
	Timeout        time.Duration
	MaxPoolSize    uint64
	MinPoolSize    uint64
}

// MongoStorage реализует интерфейс Storage для MongoDB
type MongoStorage struct {
	client     *mongo.Client
	database   *mongo.Database
	collection *mongo.Collection
	logger     *logrus.Logger
}

// New создает новое подключение к MongoDB
func New(cfg *Config, logger *logrus.Logger) (*MongoStorage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	// Настройка опций клиента
	clientOptions := options.Client().
		ApplyURI(cfg.URI).
		SetMaxPoolSize(cfg.MaxPoolSize).
		SetMinPoolSize(cfg.MinPoolSize).
		SetServerSelectionTimeout(cfg.Timeout)

	// Подключение к MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Проверка подключения
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	logger.Infof("Successfully connected to MongoDB: %s", cfg.URI)

	// Получение ссылок на базу и коллекцию
	database := client.Database(cfg.Database)
	collection := database.Collection(cfg.Collection)

	storage := &MongoStorage{
		client:     client,
		database:   database,
		collection: collection,
		logger:     logger,
	}

	// Создание индексов
	if err := storage.createIndexes(ctx); err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return storage, nil
}

// createIndexes создает необходимые индексы
func (s *MongoStorage) createIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys: map[string]interface{}{
				"user_id": 1,
			},
		},
		{
			Keys: map[string]interface{}{
				"timestamp": -1,
			},
		},
		{
			Keys: map[string]interface{}{
				"processed_at": -1,
			},
		},
		{
			Keys: map[string]interface{}{
				"type": 1,
			},
		},
		{
			Keys: map[string]interface{}{
				"status": 1,
			},
		},
		{
			Keys: map[string]interface{}{
				"amount": -1,
			},
		},
	}

	indexNames, err := s.collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	s.logger.Infof("Created %d indexes: %v", len(indexNames), indexNames)
	return nil
}

// Ping проверяет соединение с базой данных
func (s *MongoStorage) Ping(ctx context.Context) error {
	return s.client.Ping(ctx, readpref.Primary())
}

// Close закрывает соединение с базой данных
func (s *MongoStorage) Close(ctx context.Context) error {
	if s.client != nil {
		s.logger.Info("Closing MongoDB connection")
		return s.client.Disconnect(ctx)
	}
	return nil
}
