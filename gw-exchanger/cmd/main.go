package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gw-exchanger/internal/config"
	"gw-exchanger/internal/grpc"
	"gw-exchanger/internal/logger"
	"gw-exchanger/internal/storages/postgres"
	pb "gw-exchanger/proto"
	"github.com/sirupsen/logrus"
	grpcServer "google.golang.org/grpc"
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
	log.Info("Starting gw-exchanger service...")
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

	// Создание gRPC сервера
	grpcSrv := grpcServer.NewServer(
		grpcServer.UnaryInterceptor(loggingInterceptor(log)),
	)

	exchangeServer := grpc.NewExchangeServer(storage, log)
	pb.RegisterExchangeServiceServer(grpcSrv, exchangeServer)

	// Создание listener для gRPC
	listener, err := net.Listen("tcp", ":"+cfg.Server.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to create listener: %v", err)
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Запуск gRPC сервера в горутине
	go func() {
		log.Infof("gRPC server is listening on port %s", cfg.Server.GRPCPort)
		if err := grpcSrv.Serve(listener); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// Ожидание сигнала завершения
	<-done
	log.Info("Shutting down server...")

	// Graceful shutdown
	grpcSrv.GracefulStop()
	log.Info("Server stopped gracefully")
}

// loggingInterceptor создает interceptor для логирования gRPC запросов
func loggingInterceptor(log *logrus.Logger) grpcServer.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpcServer.UnaryServerInfo,
		handler grpcServer.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// Вызов обработчика
		resp, err := handler(ctx, req)

		// Логирование
		duration := time.Since(start)
		if err != nil {
			log.Errorf("gRPC method: %s, duration: %v, error: %v", info.FullMethod, duration, err)
		} else {
			log.Infof("gRPC method: %s, duration: %v, status: success", info.FullMethod, duration)
		}

		return resp, err
	}
}
