package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gw-notification/internal/config"
	"gw-notification/internal/kafka"
	"gw-notification/internal/logger"
	"gw-notification/internal/storages/mongodb"
	"gw-notification/pkg"
	"github.com/sirupsen/logrus"
)

func main() {
	// Парсинг флагов командной строки
	configPath := flag.String("c", "", "Path to config file")
	flag.Parse()

	// Загрузка конфигурации
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Валидация конфигурации
	if err := cfg.Validate(); err != nil {
		fmt.Printf("Invalid config: %v\n", err)
		os.Exit(1)
	}

	// Инициализация логгера
	log := logger.New(cfg.Logger.Level)
	log.Infof("Starting %s service...", cfg.Service.Name)
	log.Infof("Configuration loaded from: %s", *configPath)

	// Подключение к MongoDB
	mongoConfig := &mongodb.Config{
		URI:         cfg.MongoDB.URI,
		Database:    cfg.MongoDB.Database,
		Collection:  cfg.MongoDB.Collection,
		Timeout:     cfg.MongoDB.Timeout,
		MaxPoolSize: cfg.MongoDB.MaxPoolSize,
		MinPoolSize: cfg.MongoDB.MinPoolSize,
	}

	storage, err := mongodb.New(mongoConfig, log)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		storage.Close(ctx)
	}()

	// Проверка подключения к MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := storage.Ping(ctx); err != nil {
		cancel()
		log.Fatalf("MongoDB ping failed: %v", err)
	}
	cancel()
	log.Info("MongoDB connection established")

	// Создание Kafka consumer
	kafkaConfig := &kafka.Config{
		Brokers:       cfg.Kafka.Brokers,
		Topic:         cfg.Kafka.Topic,
		GroupID:       cfg.Kafka.GroupID,
		Partition:     cfg.Kafka.Partition,
		MinBytes:      cfg.Kafka.MinBytes,
		MaxBytes:      cfg.Kafka.MaxBytes,
		MaxWait:       cfg.Kafka.MaxWait,
		BatchSize:     cfg.Processing.BatchSize,
		Workers:       cfg.Processing.Workers,
		FlushInterval: cfg.Processing.FlushInterval,
		RetryAttempts: cfg.Processing.RetryAttempts,
		RetryDelay:    cfg.Processing.RetryDelay,
	}

	consumer := kafka.NewConsumer(kafkaConfig, storage, log)
	defer consumer.Close()

	// Контекст для graceful shutdown
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	// Обработка сигналов завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Запуск consumer в горутине
	consumerErr := make(chan error, 1)
	go func() {
		consumerErr <- consumer.Start(ctx)
	}()

	// Запуск горутины для вывода статистики
	statsTicker := time.NewTicker(30 * time.Second)
	defer statsTicker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-statsTicker.C:
				printStatistics(log, consumer, storage)
			}
		}
	}()

	log.Info("Service is running. Press Ctrl+C to stop...")

	// Ожидание сигнала завершения или ошибки
	select {
	case <-sigChan:
		log.Info("Received shutdown signal...")
	case err := <-consumerErr:
		if err != nil {
			log.Errorf("Consumer error: %v", err)
		}
	}

	// Graceful shutdown
	log.Info("Shutting down service...")
	cancel()

	// Даем время на завершение обработки
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Processing.MaxProcessingTime)
	defer shutdownCancel()

	// Ждем завершения consumer
	select {
	case <-shutdownCtx.Done():
		log.Warn("Shutdown timeout exceeded, forcing exit")
	case err := <-consumerErr:
		if err != nil && err != context.Canceled {
			log.Errorf("Consumer shutdown error: %v", err)
		}
	}

	// Финальная статистика
	printFinalStatistics(log, consumer, storage)

	log.Info("Service stopped gracefully")
}

// printStatistics выводит текущую статистику
func printStatistics(log *logrus.Logger, consumer *kafka.Consumer, storage *mongodb.MongoStorage) {
	// Статистика consumer
	consumerStats := consumer.GetStatistics()

	log.Infof("Consumer Statistics: Processed=%d, Failed=%d, Rate=%.2f msg/s, Uptime=%.0fs",
		consumerStats["messages_processed"],
		consumerStats["messages_failed"],
		consumerStats["processing_rate"],
		consumerStats["uptime_seconds"])

	// Статистика хранилища
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	storageStats, err := storage.GetStatistics(ctx)
	if err != nil {
		log.Warnf("Failed to get storage statistics: %v", err)
		return
	}

	log.Infof("Storage Statistics: Total=%d, Failed=%d, AvgAmount=%.2f, TotalAmount=%.2f",
		storageStats.TotalProcessed,
		storageStats.TotalFailed,
		storageStats.AverageAmount,
		storageStats.TotalAmount)
}

// printFinalStatistics выводит финальную статистику перед завершением
func printFinalStatistics(log *logrus.Logger, consumer *kafka.Consumer, storage *mongodb.MongoStorage) {
	log.Info("=== Final Statistics ===")

	consumerStats := consumer.GetStatistics()
	duration := pkg.FormatDuration(time.Duration(consumerStats["uptime_seconds"].(float64) * float64(time.Second)))

	log.Infof("Total Messages Processed: %d", consumerStats["messages_processed"])
	log.Infof("Total Messages Failed: %d", consumerStats["messages_failed"])
	log.Infof("Average Processing Rate: %.2f msg/s", consumerStats["processing_rate"])
	log.Infof("Total Uptime: %s", duration)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	storageStats, err := storage.GetStatistics(ctx)
	if err != nil {
		log.Warnf("Failed to get final storage statistics: %v", err)
		return
	}

	log.Infof("Total Transfers in DB: %d", storageStats.TotalProcessed)
	log.Infof("Average Transfer Amount: %.2f", storageStats.AverageAmount)
	log.Infof("Total Amount Processed: %.2f", storageStats.TotalAmount)
	log.Info("========================")
}
