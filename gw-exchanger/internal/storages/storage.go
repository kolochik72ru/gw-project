package storages

import "context"

// Storage определяет интерфейс для работы с хранилищем данных
// Это позволяет легко заменить PostgreSQL на другую БД
type Storage interface {
	// GetExchangeRate возвращает курс обмена для конкретной пары валют
	GetExchangeRate(ctx context.Context, fromCurrency, toCurrency string) (*ExchangeRate, error)

	// GetAllExchangeRates возвращает все курсы обмена
	GetAllExchangeRates(ctx context.Context) ([]ExchangeRate, error)

	// UpdateExchangeRate обновляет курс обмена
	UpdateExchangeRate(ctx context.Context, rate *ExchangeRate) error

	// CreateExchangeRate создает новый курс обмена
	CreateExchangeRate(ctx context.Context, rate *ExchangeRate) error

	// Close закрывает соединение с БД
	Close() error

	// Ping проверяет соединение с БД
	Ping(ctx context.Context) error
}
