package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gw-currency-wallet/internal/storages"
)

// CreateTransaction создает новую транзакцию
func (s *PostgresStorage) CreateTransaction(ctx context.Context, tx *storages.Transaction) error {
	query := `
		INSERT INTO transactions (user_id, type, from_currency, to_currency, from_amount, to_amount, exchange_rate, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	now := time.Now()
	err := s.db.QueryRowContext(ctx, query,
		tx.UserID,
		tx.Type,
		tx.FromCurrency,
		tx.ToCurrency,
		tx.FromAmount,
		tx.ToAmount,
		tx.ExchangeRate,
		tx.Status,
		now,
	).Scan(&tx.ID)

	if err != nil {
		s.logger.Errorf("Failed to create transaction: %v", err)
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	tx.CreatedAt = now

	s.logger.Infof("Created transaction: ID=%d, Type=%s, User=%d", tx.ID, tx.Type, tx.UserID)
	return nil
}

// GetTransaction возвращает транзакцию по ID
func (s *PostgresStorage) GetTransaction(ctx context.Context, txID int64) (*storages.Transaction, error) {
	query := `
		SELECT id, user_id, type, from_currency, to_currency, from_amount, to_amount, exchange_rate, status, created_at, completed_at
		FROM transactions
		WHERE id = $1
	`

	var tx storages.Transaction
	err := s.db.QueryRowContext(ctx, query, txID).Scan(
		&tx.ID,
		&tx.UserID,
		&tx.Type,
		&tx.FromCurrency,
		&tx.ToCurrency,
		&tx.FromAmount,
		&tx.ToAmount,
		&tx.ExchangeRate,
		&tx.Status,
		&tx.CreatedAt,
		&tx.CompletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("transaction not found")
	}

	if err != nil {
		s.logger.Errorf("Failed to get transaction: %v", err)
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return &tx, nil
}

// GetUserTransactions возвращает транзакции пользователя
func (s *PostgresStorage) GetUserTransactions(ctx context.Context, userID int64, limit int) ([]storages.Transaction, error) {
	query := `
		SELECT id, user_id, type, from_currency, to_currency, from_amount, to_amount, exchange_rate, status, created_at, completed_at
		FROM transactions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		s.logger.Errorf("Failed to query transactions: %v", err)
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()

	var transactions []storages.Transaction
	for rows.Next() {
		var tx storages.Transaction
		err := rows.Scan(
			&tx.ID,
			&tx.UserID,
			&tx.Type,
			&tx.FromCurrency,
			&tx.ToCurrency,
			&tx.FromAmount,
			&tx.ToAmount,
			&tx.ExchangeRate,
			&tx.Status,
			&tx.CreatedAt,
			&tx.CompletedAt,
		)
		if err != nil {
			s.logger.Errorf("Failed to scan transaction: %v", err)
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		transactions = append(transactions, tx)
	}

	if err = rows.Err(); err != nil {
		s.logger.Errorf("Error iterating transactions: %v", err)
		return nil, fmt.Errorf("error iterating transactions: %w", err)
	}

	return transactions, nil
}

// UpdateTransactionStatus обновляет статус транзакции
func (s *PostgresStorage) UpdateTransactionStatus(ctx context.Context, txID int64, status string) error {
	query := `
		UPDATE transactions
		SET status = $1, completed_at = $2
		WHERE id = $3
	`

	var completedAt *time.Time
	if status == storages.TransactionStatusCompleted || status == storages.TransactionStatusFailed {
		now := time.Now()
		completedAt = &now
	}

	result, err := s.db.ExecContext(ctx, query, status, completedAt, txID)
	if err != nil {
		s.logger.Errorf("Failed to update transaction status: %v", err)
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("transaction not found")
	}

	s.logger.Debugf("Updated transaction %d status to %s", txID, status)
	return nil
}

// ExecuteExchange выполняет обмен валюты атомарно
func (s *PostgresStorage) ExecuteExchange(ctx context.Context, userID int64, fromCurrency, toCurrency string, fromAmount, toAmount, rate float64) error {
	// Начинаем транзакцию
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		s.logger.Errorf("Failed to begin transaction: %v", err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Получаем баланс исходной валюты с блокировкой строки
	var fromBalance float64
	err = tx.QueryRowContext(ctx, `
		SELECT amount FROM balances 
		WHERE user_id = $1 AND currency = $2
		FOR UPDATE
	`, userID, fromCurrency).Scan(&fromBalance)

	if err != nil {
		s.logger.Errorf("Failed to get from balance: %v", err)
		return fmt.Errorf("failed to get balance: %w", err)
	}

	// 2. Проверяем достаточность средств
	if fromBalance < fromAmount {
		return fmt.Errorf("insufficient funds: have %.2f, need %.2f", fromBalance, fromAmount)
	}

	// 3. Уменьшаем баланс исходной валюты
	_, err = tx.ExecContext(ctx, `
		UPDATE balances
		SET amount = amount - $1, updated_at = $2
		WHERE user_id = $3 AND currency = $4
	`, fromAmount, time.Now(), userID, fromCurrency)

	if err != nil {
		s.logger.Errorf("Failed to deduct from balance: %v", err)
		return fmt.Errorf("failed to deduct balance: %w", err)
	}

	// 4. Увеличиваем баланс целевой валюты
	_, err = tx.ExecContext(ctx, `
		UPDATE balances
		SET amount = amount + $1, updated_at = $2
		WHERE user_id = $3 AND currency = $4
	`, toAmount, time.Now(), userID, toCurrency)

	if err != nil {
		s.logger.Errorf("Failed to add to balance: %v", err)
		return fmt.Errorf("failed to add balance: %w", err)
	}

	// 5. Создаем запись о транзакции
	now := time.Now()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO transactions (user_id, type, from_currency, to_currency, from_amount, to_amount, exchange_rate, status, created_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, userID, storages.TransactionTypeExchange, fromCurrency, toCurrency, fromAmount, toAmount, rate, storages.TransactionStatusCompleted, now, now)

	if err != nil {
		s.logger.Errorf("Failed to create transaction record: %v", err)
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	// 6. Коммитим транзакцию
	if err := tx.Commit(); err != nil {
		s.logger.Errorf("Failed to commit transaction: %v", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Infof("Exchange completed: User=%d, %.2f %s -> %.2f %s (rate: %.8f)",
		userID, fromAmount, fromCurrency, toAmount, toCurrency, rate)

	return nil
}
