package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gw-currency-wallet/internal/storages"
)

// CreateUser создает нового пользователя
func (s *PostgresStorage) CreateUser(ctx context.Context, user *storages.User) error {
	query := `
		INSERT INTO users (username, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	now := time.Now()
	err := s.db.QueryRowContext(ctx, query,
		user.Username,
		user.Email,
		user.PasswordHash,
		now,
		now,
	).Scan(&user.ID)

	if err != nil {
		s.logger.Errorf("Failed to create user: %v", err)
		return fmt.Errorf("failed to create user: %w", err)
	}

	user.CreatedAt = now
	user.UpdatedAt = now

	// Создаем начальные балансы для всех валют (0.0)
	currencies := []string{"USD", "EUR", "RUB"}
	for _, currency := range currencies {
		balance := &storages.Balance{
			UserID:   user.ID,
			Currency: currency,
			Amount:   0.0,
		}
		if err := s.CreateBalance(ctx, balance); err != nil {
			s.logger.Errorf("Failed to create initial balance for %s: %v", currency, err)
			return fmt.Errorf("failed to create initial balance: %w", err)
		}
	}

	s.logger.Infof("Created user: %s (ID: %d)", user.Username, user.ID)
	return nil
}

// GetUserByUsername возвращает пользователя по имени
func (s *PostgresStorage) GetUserByUsername(ctx context.Context, username string) (*storages.User, error) {
	query := `
		SELECT id, username, email, password_hash, created_at, updated_at
		FROM users
		WHERE username = $1
	`

	var user storages.User
	err := s.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}

	if err != nil {
		s.logger.Errorf("Failed to get user by username: %v", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetUserByEmail возвращает пользователя по email
func (s *PostgresStorage) GetUserByEmail(ctx context.Context, email string) (*storages.User, error) {
	query := `
		SELECT id, username, email, password_hash, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user storages.User
	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}

	if err != nil {
		s.logger.Errorf("Failed to get user by email: %v", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetUserByID возвращает пользователя по ID
func (s *PostgresStorage) GetUserByID(ctx context.Context, userID int64) (*storages.User, error) {
	query := `
		SELECT id, username, email, password_hash, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user storages.User
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}

	if err != nil {
		s.logger.Errorf("Failed to get user by ID: %v", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetBalance возвращает баланс пользователя в конкретной валюте
func (s *PostgresStorage) GetBalance(ctx context.Context, userID int64, currency string) (*storages.Balance, error) {
	query := `
		SELECT id, user_id, currency, amount, updated_at, created_at
		FROM balances
		WHERE user_id = $1 AND currency = $2
	`

	var balance storages.Balance
	err := s.db.QueryRowContext(ctx, query, userID, currency).Scan(
		&balance.ID,
		&balance.UserID,
		&balance.Currency,
		&balance.Amount,
		&balance.UpdatedAt,
		&balance.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("balance not found")
	}

	if err != nil {
		s.logger.Errorf("Failed to get balance: %v", err)
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	return &balance, nil
}

// GetAllBalances возвращает все балансы пользователя
func (s *PostgresStorage) GetAllBalances(ctx context.Context, userID int64) ([]storages.Balance, error) {
	query := `
		SELECT id, user_id, currency, amount, updated_at, created_at
		FROM balances
		WHERE user_id = $1
		ORDER BY currency
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		s.logger.Errorf("Failed to query balances: %v", err)
		return nil, fmt.Errorf("failed to query balances: %w", err)
	}
	defer rows.Close()

	var balances []storages.Balance
	for rows.Next() {
		var balance storages.Balance
		err := rows.Scan(
			&balance.ID,
			&balance.UserID,
			&balance.Currency,
			&balance.Amount,
			&balance.UpdatedAt,
			&balance.CreatedAt,
		)
		if err != nil {
			s.logger.Errorf("Failed to scan balance: %v", err)
			return nil, fmt.Errorf("failed to scan balance: %w", err)
		}
		balances = append(balances, balance)
	}

	if err = rows.Err(); err != nil {
		s.logger.Errorf("Error iterating balances: %v", err)
		return nil, fmt.Errorf("error iterating balances: %w", err)
	}

	return balances, nil
}

// UpdateBalance обновляет баланс пользователя
func (s *PostgresStorage) UpdateBalance(ctx context.Context, balance *storages.Balance) error {
	query := `
		UPDATE balances
		SET amount = $1, updated_at = $2
		WHERE user_id = $3 AND currency = $4
	`

	result, err := s.db.ExecContext(ctx, query,
		balance.Amount,
		time.Now(),
		balance.UserID,
		balance.Currency,
	)

	if err != nil {
		s.logger.Errorf("Failed to update balance: %v", err)
		return fmt.Errorf("failed to update balance: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("balance not found")
	}

	s.logger.Debugf("Updated balance for user %d, %s: %.2f", balance.UserID, balance.Currency, balance.Amount)
	return nil
}

// CreateBalance создает новый баланс
func (s *PostgresStorage) CreateBalance(ctx context.Context, balance *storages.Balance) error {
	query := `
		INSERT INTO balances (user_id, currency, amount, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	now := time.Now()
	err := s.db.QueryRowContext(ctx, query,
		balance.UserID,
		balance.Currency,
		balance.Amount,
		now,
		now,
	).Scan(&balance.ID)

	if err != nil {
		s.logger.Errorf("Failed to create balance: %v", err)
		return fmt.Errorf("failed to create balance: %w", err)
	}

	balance.CreatedAt = now
	balance.UpdatedAt = now

	s.logger.Debugf("Created balance for user %d, %s: %.2f", balance.UserID, balance.Currency, balance.Amount)
	return nil
}
