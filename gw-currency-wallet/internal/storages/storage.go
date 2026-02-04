package storages

import "context"

// Storage определяет интерфейс для работы с хранилищем данных
type Storage interface {
	// User operations
	CreateUser(ctx context.Context, user *User) error
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, userID int64) (*User, error)
	
	// Balance operations
	GetBalance(ctx context.Context, userID int64, currency string) (*Balance, error)
	GetAllBalances(ctx context.Context, userID int64) ([]Balance, error)
	UpdateBalance(ctx context.Context, balance *Balance) error
	CreateBalance(ctx context.Context, balance *Balance) error
	
	// Transaction operations
	CreateTransaction(ctx context.Context, tx *Transaction) error
	GetTransaction(ctx context.Context, txID int64) (*Transaction, error)
	GetUserTransactions(ctx context.Context, userID int64, limit int) ([]Transaction, error)
	UpdateTransactionStatus(ctx context.Context, txID int64, status string) error
	
	// Atomic operations for exchange
	ExecuteExchange(ctx context.Context, userID int64, fromCurrency, toCurrency string, fromAmount, toAmount, rate float64) error
	
	// Health check
	Ping(ctx context.Context) error
	Close() error
}
