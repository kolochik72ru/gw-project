package storages

import "time"

// User представляет пользователя системы
type User struct {
	ID           int64     `db:"id"`
	Username     string    `db:"username"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

// Balance представляет баланс пользователя в определенной валюте
type Balance struct {
	ID        int64     `db:"id"`
	UserID    int64     `db:"user_id"`
	Currency  string    `db:"currency"`
	Amount    float64   `db:"amount"`
	UpdatedAt time.Time `db:"updated_at"`
	CreatedAt time.Time `db:"created_at"`
}

// Transaction представляет транзакцию (пополнение, вывод, обмен)
type Transaction struct {
	ID              int64     `db:"id"`
	UserID          int64     `db:"user_id"`
	Type            string    `db:"type"` // deposit, withdraw, exchange
	FromCurrency    string    `db:"from_currency"`
	ToCurrency      string    `db:"to_currency"`
	FromAmount      float64   `db:"from_amount"`
	ToAmount        float64   `db:"to_amount"`
	ExchangeRate    float64   `db:"exchange_rate"`
	Status          string    `db:"status"` // pending, completed, failed
	CreatedAt       time.Time `db:"created_at"`
	CompletedAt     *time.Time `db:"completed_at"`
}

// TransactionType определяет типы транзакций
const (
	TransactionTypeDeposit  = "deposit"
	TransactionTypeWithdraw = "withdraw"
	TransactionTypeExchange = "exchange"
)

// TransactionStatus определяет статусы транзакций
const (
	TransactionStatusPending   = "pending"
	TransactionStatusCompleted = "completed"
	TransactionStatusFailed    = "failed"
)

// UserBalances представляет балансы пользователя во всех валютах
type UserBalances struct {
	USD float64 `json:"USD"`
	EUR float64 `json:"EUR"`
	RUB float64 `json:"RUB"`
}
