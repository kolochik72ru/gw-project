package storages

import "context"

// Storage определяет интерфейс для работы с хранилищем данных
type Storage interface {
	// SaveTransfer сохраняет информацию о крупном переводе
	SaveTransfer(ctx context.Context, transfer *LargeTransfer) error

	// SaveTransferBatch сохраняет пакет переводов
	SaveTransferBatch(ctx context.Context, transfers []LargeTransfer) error

	// GetTransfer получает перевод по ID
	GetTransfer(ctx context.Context, id string) (*LargeTransfer, error)

	// GetTransfersByUser получает переводы пользователя
	GetTransfersByUser(ctx context.Context, userID int64, limit int) ([]LargeTransfer, error)

	// GetRecentTransfers получает последние переводы
	GetRecentTransfers(ctx context.Context, limit int) ([]LargeTransfer, error)

	// GetStatistics возвращает статистику обработки
	GetStatistics(ctx context.Context) (*Statistics, error)

	// Health check
	Ping(ctx context.Context) error
	Close(ctx context.Context) error
}
