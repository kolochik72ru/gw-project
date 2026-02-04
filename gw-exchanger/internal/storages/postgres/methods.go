package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gw-exchanger/internal/storages"
)

// GetExchangeRate возвращает курс обмена для конкретной пары валют
func (s *PostgresStorage) GetExchangeRate(ctx context.Context, fromCurrency, toCurrency string) (*storages.ExchangeRate, error) {
	query := `
		SELECT id, from_currency, to_currency, rate, updated_at, created_at
		FROM exchange_rates
		WHERE from_currency = $1 AND to_currency = $2
	`

	var rate storages.ExchangeRate
	err := s.db.QueryRowContext(ctx, query, fromCurrency, toCurrency).Scan(
		&rate.ID,
		&rate.FromCurrency,
		&rate.ToCurrency,
		&rate.Rate,
		&rate.UpdatedAt,
		&rate.CreatedAt,
	)

	if err == sql.ErrNoRows {
		s.logger.Warnf("Exchange rate not found: %s -> %s", fromCurrency, toCurrency)
		return nil, fmt.Errorf("exchange rate not found for %s to %s", fromCurrency, toCurrency)
	}

	if err != nil {
		s.logger.Errorf("Failed to get exchange rate: %v", err)
		return nil, fmt.Errorf("failed to get exchange rate: %w", err)
	}

	s.logger.Debugf("Retrieved exchange rate: %s -> %s = %.8f", fromCurrency, toCurrency, rate.Rate)
	return &rate, nil
}

// GetAllExchangeRates возвращает все курсы обмена
func (s *PostgresStorage) GetAllExchangeRates(ctx context.Context) ([]storages.ExchangeRate, error) {
	query := `
		SELECT id, from_currency, to_currency, rate, updated_at, created_at
		FROM exchange_rates
		ORDER BY from_currency, to_currency
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		s.logger.Errorf("Failed to query exchange rates: %v", err)
		return nil, fmt.Errorf("failed to query exchange rates: %w", err)
	}
	defer rows.Close()

	var rates []storages.ExchangeRate
	for rows.Next() {
		var rate storages.ExchangeRate
		err := rows.Scan(
			&rate.ID,
			&rate.FromCurrency,
			&rate.ToCurrency,
			&rate.Rate,
			&rate.UpdatedAt,
			&rate.CreatedAt,
		)
		if err != nil {
			s.logger.Errorf("Failed to scan exchange rate: %v", err)
			return nil, fmt.Errorf("failed to scan exchange rate: %w", err)
		}
		rates = append(rates, rate)
	}

	if err = rows.Err(); err != nil {
		s.logger.Errorf("Error iterating exchange rates: %v", err)
		return nil, fmt.Errorf("error iterating exchange rates: %w", err)
	}

	s.logger.Debugf("Retrieved %d exchange rates", len(rates))
	return rates, nil
}

// UpdateExchangeRate обновляет существующий курс обмена
func (s *PostgresStorage) UpdateExchangeRate(ctx context.Context, rate *storages.ExchangeRate) error {
	query := `
		UPDATE exchange_rates
		SET rate = $1, updated_at = $2
		WHERE from_currency = $3 AND to_currency = $4
	`

	result, err := s.db.ExecContext(ctx, query,
		rate.Rate,
		time.Now(),
		rate.FromCurrency,
		rate.ToCurrency,
	)

	if err != nil {
		s.logger.Errorf("Failed to update exchange rate: %v", err)
		return fmt.Errorf("failed to update exchange rate: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		s.logger.Warnf("No rows updated for %s -> %s", rate.FromCurrency, rate.ToCurrency)
		return fmt.Errorf("exchange rate not found for %s to %s", rate.FromCurrency, rate.ToCurrency)
	}

	s.logger.Infof("Updated exchange rate: %s -> %s = %.8f", rate.FromCurrency, rate.ToCurrency, rate.Rate)
	return nil
}

// CreateExchangeRate создает новый курс обмена
func (s *PostgresStorage) CreateExchangeRate(ctx context.Context, rate *storages.ExchangeRate) error {
	query := `
		INSERT INTO exchange_rates (from_currency, to_currency, rate, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	now := time.Now()
	err := s.db.QueryRowContext(ctx, query,
		rate.FromCurrency,
		rate.ToCurrency,
		rate.Rate,
		now,
		now,
	).Scan(&rate.ID)

	if err != nil {
		s.logger.Errorf("Failed to create exchange rate: %v", err)
		return fmt.Errorf("failed to create exchange rate: %w", err)
	}

	rate.CreatedAt = now
	rate.UpdatedAt = now

	s.logger.Infof("Created exchange rate: %s -> %s = %.8f (ID: %d)",
		rate.FromCurrency, rate.ToCurrency, rate.Rate, rate.ID)
	return nil
}
