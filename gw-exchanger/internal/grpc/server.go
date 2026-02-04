package grpc

import (
	"context"
	"fmt"

	"gw-exchanger/internal/storages"
	pb "gw-exchanger/proto"
	"github.com/sirupsen/logrus"
)

// ExchangeServer реализует gRPC сервис ExchangeService
type ExchangeServer struct {
	pb.UnimplementedExchangeServiceServer
	storage storages.Storage
	logger  *logrus.Logger
}

// NewExchangeServer создает новый экземпляр ExchangeServer
func NewExchangeServer(storage storages.Storage, logger *logrus.Logger) *ExchangeServer {
	return &ExchangeServer{
		storage: storage,
		logger:  logger,
	}
}

// GetExchangeRates возвращает все курсы обмена валют
func (s *ExchangeServer) GetExchangeRates(ctx context.Context, req *pb.Empty) (*pb.ExchangeRatesResponse, error) {
	s.logger.Info("Received GetExchangeRates request")

	rates, err := s.storage.GetAllExchangeRates(ctx)
	if err != nil {
		s.logger.Errorf("Failed to get exchange rates: %v", err)
		return nil, fmt.Errorf("failed to get exchange rates: %w", err)
	}

	// Преобразование данных из БД в формат protobuf
	ratesMap := make(map[string]float32)
	for _, rate := range rates {
		key := fmt.Sprintf("%s_%s", rate.FromCurrency, rate.ToCurrency)
		ratesMap[key] = float32(rate.Rate)
	}

	response := &pb.ExchangeRatesResponse{
		Rates: ratesMap,
	}

	s.logger.Infof("Successfully retrieved %d exchange rates", len(rates))
	return response, nil
}

// GetExchangeRateForCurrency возвращает курс обмена для конкретной пары валют
func (s *ExchangeServer) GetExchangeRateForCurrency(ctx context.Context, req *pb.CurrencyRequest) (*pb.ExchangeRateResponse, error) {
	s.logger.Infof("Received GetExchangeRateForCurrency request: %s -> %s",
		req.FromCurrency, req.ToCurrency)

	// Валидация входных данных
	if req.FromCurrency == "" || req.ToCurrency == "" {
		s.logger.Warn("Invalid currency request: empty currency code")
		return nil, fmt.Errorf("from_currency and to_currency are required")
	}

	// Проверка, что валюты разные
	if req.FromCurrency == req.ToCurrency {
		s.logger.Warnf("Same currency conversion requested: %s", req.FromCurrency)
		return &pb.ExchangeRateResponse{
			FromCurrency: req.FromCurrency,
			ToCurrency:   req.ToCurrency,
			Rate:         1.0,
		}, nil
	}

	// Получение курса из БД
	rate, err := s.storage.GetExchangeRate(ctx, req.FromCurrency, req.ToCurrency)
	if err != nil {
		s.logger.Errorf("Failed to get exchange rate for %s -> %s: %v",
			req.FromCurrency, req.ToCurrency, err)
		return nil, fmt.Errorf("exchange rate not found: %w", err)
	}

	response := &pb.ExchangeRateResponse{
		FromCurrency: rate.FromCurrency,
		ToCurrency:   rate.ToCurrency,
		Rate:         float32(rate.Rate),
	}

	s.logger.Infof("Successfully retrieved exchange rate: %s -> %s = %.8f",
		rate.FromCurrency, rate.ToCurrency, rate.Rate)

	return response, nil
}
