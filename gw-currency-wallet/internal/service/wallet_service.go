package service

import (
	"context"
	"fmt"

	"gw-currency-wallet/internal/cache"
	"gw-currency-wallet/internal/grpc"
	"gw-currency-wallet/internal/kafka"
	"gw-currency-wallet/internal/storages"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// WalletService сервисный слой для бизнес-логики
type WalletService struct {
	storage         storages.Storage
	exchangerClient *grpc.ExchangerClient
	ratesCache      *cache.RatesCache
	kafkaProducer   *kafka.Producer
	logger          *logrus.Logger
}

// NewWalletService создает новый экземпляр сервиса
func NewWalletService(
	storage storages.Storage,
	exchangerClient *grpc.ExchangerClient,
	ratesCache *cache.RatesCache,
	kafkaProducer *kafka.Producer,
	logger *logrus.Logger,
) *WalletService {
	return &WalletService{
		storage:         storage,
		exchangerClient: exchangerClient,
		ratesCache:      ratesCache,
		kafkaProducer:   kafkaProducer,
		logger:          logger,
	}
}

// RegisterUser регистрирует нового пользователя
func (s *WalletService) RegisterUser(ctx context.Context, username, email, password string) error {
	// Проверяем, не существует ли уже пользователь
	existingUser, _ := s.storage.GetUserByUsername(ctx, username)
	if existingUser != nil {
		return fmt.Errorf("username already exists")
	}

	existingUser, _ = s.storage.GetUserByEmail(ctx, email)
	if existingUser != nil {
		return fmt.Errorf("email already exists")
	}

	// Хешируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Errorf("Failed to hash password: %v", err)
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Создаем пользователя
	user := &storages.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedPassword),
	}

	if err := s.storage.CreateUser(ctx, user); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.Infof("User registered successfully: %s", username)
	return nil
}

// AuthenticateUser аутентифицирует пользователя
func (s *WalletService) AuthenticateUser(ctx context.Context, username, password string) (*storages.User, error) {
	user, err := s.storage.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("invalid username or password")
	}

	// Проверяем пароль
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.logger.Warnf("Failed authentication attempt for user: %s", username)
		return nil, fmt.Errorf("invalid username or password")
	}

	s.logger.Infof("User authenticated successfully: %s", username)
	return user, nil
}

// GetUserBalances возвращает балансы пользователя
func (s *WalletService) GetUserBalances(ctx context.Context, userID int64) (*storages.UserBalances, error) {
	balances, err := s.storage.GetAllBalances(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get balances: %w", err)
	}

	userBalances := &storages.UserBalances{}
	for _, balance := range balances {
		switch balance.Currency {
		case "USD":
			userBalances.USD = balance.Amount
		case "EUR":
			userBalances.EUR = balance.Amount
		case "RUB":
			userBalances.RUB = balance.Amount
		}
	}

	return userBalances, nil
}

// Deposit пополняет баланс пользователя
func (s *WalletService) Deposit(ctx context.Context, userID int64, currency string, amount float64) (*storages.UserBalances, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	// Получаем текущий баланс
	balance, err := s.storage.GetBalance(ctx, userID, currency)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	// Обновляем баланс
	balance.Amount += amount
	if err := s.storage.UpdateBalance(ctx, balance); err != nil {
		return nil, fmt.Errorf("failed to update balance: %w", err)
	}

	// Создаем запись о транзакции
	tx := &storages.Transaction{
		UserID:       userID,
		Type:         storages.TransactionTypeDeposit,
		FromCurrency: currency,
		ToCurrency:   currency,
		FromAmount:   amount,
		ToAmount:     amount,
		ExchangeRate: 1.0,
		Status:       storages.TransactionStatusCompleted,
	}
	if err := s.storage.CreateTransaction(ctx, tx); err != nil {
		s.logger.Warnf("Failed to create transaction record: %v", err)
	}

	// Отправляем уведомление в Kafka, если сумма большая
	if err := s.kafkaProducer.SendLargeTransferNotification(ctx, userID, "deposit", currency, currency, amount); err != nil {
		s.logger.Warnf("Failed to send Kafka notification: %v", err)
	}

	s.logger.Infof("Deposit completed: UserID=%d, Amount=%.2f %s", userID, amount, currency)

	return s.GetUserBalances(ctx, userID)
}

// Withdraw выводит средства со счета пользователя
func (s *WalletService) Withdraw(ctx context.Context, userID int64, currency string, amount float64) (*storages.UserBalances, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	// Получаем текущий баланс
	balance, err := s.storage.GetBalance(ctx, userID, currency)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	// Проверяем достаточность средств
	if balance.Amount < amount {
		return nil, fmt.Errorf("insufficient funds: have %.2f, need %.2f", balance.Amount, amount)
	}

	// Обновляем баланс
	balance.Amount -= amount
	if err := s.storage.UpdateBalance(ctx, balance); err != nil {
		return nil, fmt.Errorf("failed to update balance: %w", err)
	}

	// Создаем запись о транзакции
	tx := &storages.Transaction{
		UserID:       userID,
		Type:         storages.TransactionTypeWithdraw,
		FromCurrency: currency,
		ToCurrency:   currency,
		FromAmount:   amount,
		ToAmount:     amount,
		ExchangeRate: 1.0,
		Status:       storages.TransactionStatusCompleted,
	}
	if err := s.storage.CreateTransaction(ctx, tx); err != nil {
		s.logger.Warnf("Failed to create transaction record: %v", err)
	}

	// Отправляем уведомление в Kafka, если сумма большая
	if err := s.kafkaProducer.SendLargeTransferNotification(ctx, userID, "withdraw", currency, currency, amount); err != nil {
		s.logger.Warnf("Failed to send Kafka notification: %v", err)
	}

	s.logger.Infof("Withdrawal completed: UserID=%d, Amount=%.2f %s", userID, amount, currency)

	return s.GetUserBalances(ctx, userID)
}

// GetExchangeRates получает курсы валют (из кеша или gRPC)
func (s *WalletService) GetExchangeRates(ctx context.Context) (map[string]float32, error) {
	// Пытаемся получить из кеша
	if rates, ok := s.ratesCache.Get(); ok {
		s.logger.Debug("Returning exchange rates from cache")
		return rates, nil
	}

	// Получаем из gRPC сервиса
	s.logger.Debug("Fetching exchange rates from exchanger service")
	rates, err := s.exchangerClient.GetExchangeRates(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange rates: %w", err)
	}

	// Сохраняем в кеш
	s.ratesCache.Set(rates)

	return rates, nil
}

// ExchangeCurrency обменивает валюту
func (s *WalletService) ExchangeCurrency(ctx context.Context, userID int64, fromCurrency, toCurrency string, amount float64) (float64, *storages.UserBalances, error) {
	if amount <= 0 {
		return 0, nil, fmt.Errorf("amount must be positive")
	}

	if fromCurrency == toCurrency {
		return 0, nil, fmt.Errorf("from_currency and to_currency must be different")
	}

	// Получаем курс обмена (из кеша или gRPC)
	var rate float32
	var err error

	// Пытаемся получить из кеша
	rate, ok := s.ratesCache.GetRate(fromCurrency, toCurrency)
	if !ok {
		// Получаем из gRPC сервиса
		s.logger.Debugf("Fetching exchange rate from exchanger service: %s -> %s", fromCurrency, toCurrency)
		rate, err = s.exchangerClient.GetExchangeRateForCurrency(ctx, fromCurrency, toCurrency)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to get exchange rate: %w", err)
		}
	} else {
		s.logger.Debugf("Using cached exchange rate: %s -> %s = %.8f", fromCurrency, toCurrency, rate)
	}

	// Вычисляем сумму после обмена
	exchangedAmount := float64(rate) * amount

	// Выполняем обмен атомарно
	if err := s.storage.ExecuteExchange(ctx, userID, fromCurrency, toCurrency, amount, exchangedAmount, float64(rate)); err != nil {
		return 0, nil, fmt.Errorf("failed to execute exchange: %w", err)
	}

	// Отправляем уведомление в Kafka, если сумма большая
	if err := s.kafkaProducer.SendLargeTransferNotification(ctx, userID, "exchange", fromCurrency, toCurrency, amount); err != nil {
		s.logger.Warnf("Failed to send Kafka notification: %v", err)
	}

	s.logger.Infof("Exchange completed: UserID=%d, %.2f %s -> %.2f %s (rate: %.8f)",
		userID, amount, fromCurrency, exchangedAmount, toCurrency, rate)

	// Получаем обновленные балансы
	balances, err := s.GetUserBalances(ctx, userID)
	if err != nil {
		return exchangedAmount, nil, nil
	}

	return exchangedAmount, balances, nil
}
