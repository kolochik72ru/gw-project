package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gw-currency-wallet/internal/api"
	"gw-currency-wallet/internal/api/middleware"
	"gw-currency-wallet/internal/cache"
	"gw-currency-wallet/internal/config"
	"gw-currency-wallet/internal/grpc"
	"gw-currency-wallet/internal/kafka"
	"gw-currency-wallet/internal/logger"
	"gw-currency-wallet/internal/service"
	"gw-currency-wallet/internal/storages/postgres"
)

// @title Currency Wallet API
// @version 1.0
// @description API for currency wallet management with exchange capabilities
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

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
	log.Info("Starting gw-currency-wallet service...")
	log.Infof("Configuration loaded from: %s", *configPath)

	// Подключение к базе данных
	dbConfig := &postgres.Config{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		DBName:          cfg.Database.DBName,
		SSLMode:         cfg.Database.SSLMode,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
	}

	storage, err := postgres.New(dbConfig, log)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer storage.Close()

	// Проверка подключения к БД
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := storage.Ping(ctx); err != nil {
		cancel()
		log.Fatalf("Database ping failed: %v", err)
	}
	cancel()
	log.Info("Database connection established")

	// Подключение к gRPC exchanger service
	exchangerClient, err := grpc.NewExchangerClient(
		cfg.Exchanger.Host,
		cfg.Exchanger.Port,
		cfg.Exchanger.Timeout,
		log,
	)
	if err != nil {
		log.Fatalf("Failed to connect to exchanger service: %v", err)
	}
	defer exchangerClient.Close()

	// Проверка подключения к exchanger service
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	if err := exchangerClient.Ping(ctx); err != nil {
		cancel()
		log.Warnf("Exchanger service ping failed: %v (service may be unavailable)", err)
	} else {
		cancel()
		log.Info("Connected to exchanger service")
	}

	// Инициализация кеша курсов валют
	ratesCache := cache.NewRatesCache(cfg.Cache.RatesTTL)
	log.Info("Rates cache initialized")

	// Инициализация Kafka producer
	kafkaProducer := kafka.NewProducer(
		cfg.Kafka.Brokers,
		cfg.Kafka.Topic,
		cfg.Kafka.TransferThreshold,
		log,
	)
	defer kafkaProducer.Close()

	// Создание сервисного слоя
	walletService := service.NewWalletService(
		storage,
		exchangerClient,
		ratesCache,
		kafkaProducer,
		log,
	)
	log.Info("Wallet service initialized")

	// Создание JWT middleware
	jwtMiddleware := middleware.NewJWTMiddleware(cfg.JWT.Secret, log)

	// Настройка роутера
	router := api.SetupRouter(walletService, jwtMiddleware, log, cfg.Server.GinMode)

	// Создание HTTP сервера
	srv := &http.Server{
		Addr:         ":" + cfg.Server.HTTPPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Запуск HTTP сервера в горутине
	go func() {
		log.Infof("HTTP server is listening on port %s", cfg.Server.HTTPPort)
		log.Infof("Swagger documentation available at: http://localhost:%s/swagger/index.html", cfg.Server.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Ожидание сигнала завершения
	<-done
	log.Info("Shutting down server...")

	// Graceful shutdown с таймаутом
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Errorf("Server forced to shutdown: %v", err)
	}

	log.Info("Server stopped gracefully")
}
