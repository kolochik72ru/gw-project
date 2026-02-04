package storages

import "time"

// ExchangeRate представляет курс обмена валют
type ExchangeRate struct {
	ID           int64     `db:"id"`
	FromCurrency string    `db:"from_currency"`
	ToCurrency   string    `db:"to_currency"`
	Rate         float64   `db:"rate"`
	UpdatedAt    time.Time `db:"updated_at"`
	CreatedAt    time.Time `db:"created_at"`
}

// Currency представляет поддерживаемую валюту
type Currency struct {
	ID        int64     `db:"id"`
	Code      string    `db:"code"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
}
