package grpc

import (
	"context"
	"fmt"
	"time"

	pb "gw-currency-wallet/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ExchangerClient обертка над gRPC клиентом для exchanger сервиса
type ExchangerClient struct {
	client  pb.ExchangeServiceClient
	conn    *grpc.ClientConn
	timeout time.Duration
	logger  *logrus.Logger
}

// NewExchangerClient создает новый gRPC клиент
func NewExchangerClient(host, port string, timeout time.Duration, logger *logrus.Logger) (*ExchangerClient, error) {
	address := fmt.Sprintf("%s:%s", host, port)

	// Создаем соединение с gRPC сервером
	conn, err := grpc.Dial(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(10*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to exchanger service: %w", err)
	}

	client := pb.NewExchangeServiceClient(conn)

	logger.Infof("Connected to exchanger service at %s", address)

	return &ExchangerClient{
		client:  client,
		conn:    conn,
		timeout: timeout,
		logger:  logger,
	}, nil
}

// GetExchangeRates получает все курсы валют
func (c *ExchangerClient) GetExchangeRates(ctx context.Context) (map[string]float32, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	c.logger.Debug("Requesting exchange rates from exchanger service")

	resp, err := c.client.GetExchangeRates(ctx, &pb.Empty{})
	if err != nil {
		c.logger.Errorf("Failed to get exchange rates: %v", err)
		return nil, fmt.Errorf("failed to get exchange rates: %w", err)
	}

	c.logger.Debugf("Received %d exchange rates", len(resp.Rates))
	return resp.Rates, nil
}

// GetExchangeRateForCurrency получает курс для конкретной пары валют
func (c *ExchangerClient) GetExchangeRateForCurrency(ctx context.Context, fromCurrency, toCurrency string) (float32, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	c.logger.Debugf("Requesting exchange rate: %s -> %s", fromCurrency, toCurrency)

	req := &pb.CurrencyRequest{
		FromCurrency: fromCurrency,
		ToCurrency:   toCurrency,
	}

	resp, err := c.client.GetExchangeRateForCurrency(ctx, req)
	if err != nil {
		c.logger.Errorf("Failed to get exchange rate for %s->%s: %v", fromCurrency, toCurrency, err)
		return 0, fmt.Errorf("failed to get exchange rate: %w", err)
	}

	c.logger.Debugf("Received exchange rate: %s -> %s = %.8f", fromCurrency, toCurrency, resp.Rate)
	return resp.Rate, nil
}

// Close закрывает соединение с gRPC сервером
func (c *ExchangerClient) Close() error {
	if c.conn != nil {
		c.logger.Info("Closing connection to exchanger service")
		return c.conn.Close()
	}
	return nil
}

// Ping проверяет доступность сервиса
func (c *ExchangerClient) Ping(ctx context.Context) error {
	_, err := c.GetExchangeRates(ctx)
	return err
}
