package tests

import (
	"context"
	"testing"

	"gw-currency-wallet/internal/cache"
	"gw-currency-wallet/internal/service"
	"gw-currency-wallet/internal/storages"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"time"
)

// MockStorage - мок для Storage
type MockStorage struct {
	users    map[string]*storages.User
	balances map[int64]map[string]*storages.Balance
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		users:    make(map[string]*storages.User),
		balances: make(map[int64]map[string]*storages.Balance),
	}
}

func (m *MockStorage) CreateUser(ctx context.Context, user *storages.User) error {
	user.ID = int64(len(m.users) + 1)
	m.users[user.Username] = user
	
	// Инициализируем балансы
	m.balances[user.ID] = make(map[string]*storages.Balance)
	for _, currency := range []string{"USD", "EUR", "RUB"} {
		m.balances[user.ID][currency] = &storages.Balance{
			UserID:   user.ID,
			Currency: currency,
			Amount:   0.0,
		}
	}
	
	return nil
}

func (m *MockStorage) GetUserByUsername(ctx context.Context, username string) (*storages.User, error) {
	if user, exists := m.users[username]; exists {
		return user, nil
	}
	return nil, nil
}

func (m *MockStorage) GetUserByEmail(ctx context.Context, email string) (*storages.User, error) {
	return nil, nil
}

func (m *MockStorage) GetUserByID(ctx context.Context, userID int64) (*storages.User, error) {
	return nil, nil
}

func (m *MockStorage) GetBalance(ctx context.Context, userID int64, currency string) (*storages.Balance, error) {
	if userBalances, exists := m.balances[userID]; exists {
		if balance, exists := userBalances[currency]; exists {
			return balance, nil
		}
	}
	return nil, nil
}

func (m *MockStorage) GetAllBalances(ctx context.Context, userID int64) ([]storages.Balance, error) {
	var result []storages.Balance
	if userBalances, exists := m.balances[userID]; exists {
		for _, balance := range userBalances {
			result = append(result, *balance)
		}
	}
	return result, nil
}

func (m *MockStorage) UpdateBalance(ctx context.Context, balance *storages.Balance) error {
	if userBalances, exists := m.balances[balance.UserID]; exists {
		userBalances[balance.Currency].Amount = balance.Amount
	}
	return nil
}

func (m *MockStorage) CreateBalance(ctx context.Context, balance *storages.Balance) error {
	return nil
}

func (m *MockStorage) CreateTransaction(ctx context.Context, tx *storages.Transaction) error {
	return nil
}

func (m *MockStorage) GetTransaction(ctx context.Context, txID int64) (*storages.Transaction, error) {
	return nil, nil
}

func (m *MockStorage) GetUserTransactions(ctx context.Context, userID int64, limit int) ([]storages.Transaction, error) {
	return nil, nil
}

func (m *MockStorage) UpdateTransactionStatus(ctx context.Context, txID int64, status string) error {
	return nil
}

func (m *MockStorage) ExecuteExchange(ctx context.Context, userID int64, fromCurrency, toCurrency string, fromAmount, toAmount, rate float64) error {
	return nil
}

func (m *MockStorage) Ping(ctx context.Context) error {
	return nil
}

func (m *MockStorage) Close() error {
	return nil
}

// Tests

func TestRegisterUser(t *testing.T) {
	storage := NewMockStorage()
	ratesCache := cache.NewRatesCache(5 * time.Minute)
	logger := logrus.New()
	
	svc := service.NewWalletService(storage, nil, ratesCache, nil, logger)
	
	ctx := context.Background()
	
	// Test successful registration
	err := svc.RegisterUser(ctx, "testuser", "test@example.com", "password123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// Test duplicate username
	err = svc.RegisterUser(ctx, "testuser", "another@example.com", "password123")
	if err == nil {
		t.Fatal("Expected error for duplicate username")
	}
}

func TestAuthenticateUser(t *testing.T) {
	storage := NewMockStorage()
	ratesCache := cache.NewRatesCache(5 * time.Minute)
	logger := logrus.New()
	
	svc := service.NewWalletService(storage, nil, ratesCache, nil, logger)
	
	ctx := context.Background()
	
	// Create user
	password := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	user := &storages.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: string(hashedPassword),
	}
	storage.CreateUser(ctx, user)
	
	// Test successful authentication
	authenticatedUser, err := svc.AuthenticateUser(ctx, "testuser", password)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if authenticatedUser.Username != "testuser" {
		t.Fatalf("Expected username 'testuser', got '%s'", authenticatedUser.Username)
	}
	
	// Test failed authentication
	_, err = svc.AuthenticateUser(ctx, "testuser", "wrongpassword")
	if err == nil {
		t.Fatal("Expected error for wrong password")
	}
}

func TestDeposit(t *testing.T) {
	storage := NewMockStorage()
	ratesCache := cache.NewRatesCache(5 * time.Minute)
	logger := logrus.New()
	
	svc := service.NewWalletService(storage, nil, ratesCache, nil, logger)
	
	ctx := context.Background()
	
	// Create user
	user := &storages.User{
		Username: "testuser",
		Email:    "test@example.com",
	}
	storage.CreateUser(ctx, user)
	
	// Test deposit
	balances, err := svc.Deposit(ctx, user.ID, "USD", 100.0)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if balances.USD != 100.0 {
		t.Fatalf("Expected USD balance 100.0, got %.2f", balances.USD)
	}
	
	// Test invalid amount
	_, err = svc.Deposit(ctx, user.ID, "USD", -50.0)
	if err == nil {
		t.Fatal("Expected error for negative amount")
	}
}

func TestWithdraw(t *testing.T) {
	storage := NewMockStorage()
	ratesCache := cache.NewRatesCache(5 * time.Minute)
	logger := logrus.New()
	
	svc := service.NewWalletService(storage, nil, ratesCache, nil, logger)
	
	ctx := context.Background()
	
	// Create user and deposit
	user := &storages.User{
		Username: "testuser",
		Email:    "test@example.com",
	}
	storage.CreateUser(ctx, user)
	svc.Deposit(ctx, user.ID, "USD", 100.0)
	
	// Test successful withdrawal
	balances, err := svc.Withdraw(ctx, user.ID, "USD", 50.0)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if balances.USD != 50.0 {
		t.Fatalf("Expected USD balance 50.0, got %.2f", balances.USD)
	}
	
	// Test insufficient funds
	_, err = svc.Withdraw(ctx, user.ID, "USD", 100.0)
	if err == nil {
		t.Fatal("Expected error for insufficient funds")
	}
}
