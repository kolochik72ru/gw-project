package tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"gw-notification/internal/storages"
)

// MockStorage - мок для Storage
type MockStorage struct {
	transfers []storages.LargeTransfer
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		transfers: make([]storages.LargeTransfer, 0),
	}
}

func (m *MockStorage) SaveTransfer(ctx context.Context, transfer *storages.LargeTransfer) error {
	m.transfers = append(m.transfers, *transfer)
	return nil
}

func (m *MockStorage) SaveTransferBatch(ctx context.Context, transfers []storages.LargeTransfer) error {
	m.transfers = append(m.transfers, transfers...)
	return nil
}

func (m *MockStorage) GetTransfer(ctx context.Context, id string) (*storages.LargeTransfer, error) {
	if len(m.transfers) > 0 {
		return &m.transfers[0], nil
	}
	return nil, nil
}

func (m *MockStorage) GetTransfersByUser(ctx context.Context, userID int64, limit int) ([]storages.LargeTransfer, error) {
	var result []storages.LargeTransfer
	for _, t := range m.transfers {
		if t.UserID == userID {
			result = append(result, t)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *MockStorage) GetRecentTransfers(ctx context.Context, limit int) ([]storages.LargeTransfer, error) {
	if len(m.transfers) <= limit {
		return m.transfers, nil
	}
	return m.transfers[:limit], nil
}

func (m *MockStorage) GetStatistics(ctx context.Context) (*storages.Statistics, error) {
	stats := &storages.Statistics{
		TotalProcessed: int64(len(m.transfers)),
		TotalFailed:    0,
	}

	var totalAmount float64
	for _, t := range m.transfers {
		totalAmount += t.Amount
	}

	if len(m.transfers) > 0 {
		stats.AverageAmount = totalAmount / float64(len(m.transfers))
		stats.TotalAmount = totalAmount
		stats.LastProcessedAt = m.transfers[len(m.transfers)-1].ProcessedAt
	}

	return stats, nil
}

func (m *MockStorage) Ping(ctx context.Context) error {
	return nil
}

func (m *MockStorage) Close(ctx context.Context) error {
	return nil
}

// Tests

func TestSaveTransfer(t *testing.T) {
	storage := NewMockStorage()
	ctx := context.Background()

	transfer := &storages.LargeTransfer{
		UserID:       1,
		Type:         storages.TransferTypeDeposit,
		FromCurrency: "USD",
		ToCurrency:   "USD",
		Amount:       50000.0,
		Timestamp:    time.Now(),
	}

	err := storage.SaveTransfer(ctx, transfer)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(storage.transfers) != 1 {
		t.Fatalf("Expected 1 transfer, got %d", len(storage.transfers))
	}

	if storage.transfers[0].Amount != 50000.0 {
		t.Fatalf("Expected amount 50000.0, got %.2f", storage.transfers[0].Amount)
	}
}

func TestSaveTransferBatch(t *testing.T) {
	storage := NewMockStorage()
	ctx := context.Background()

	batch := []storages.LargeTransfer{
		{
			UserID: 1,
			Type:   storages.TransferTypeDeposit,
			Amount: 50000.0,
		},
		{
			UserID: 2,
			Type:   storages.TransferTypeExchange,
			Amount: 75000.0,
		},
		{
			UserID: 3,
			Type:   storages.TransferTypeWithdraw,
			Amount: 100000.0,
		},
	}

	err := storage.SaveTransferBatch(ctx, batch)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(storage.transfers) != 3 {
		t.Fatalf("Expected 3 transfers, got %d", len(storage.transfers))
	}
}

func TestGetTransfersByUser(t *testing.T) {
	storage := NewMockStorage()
	ctx := context.Background()

	// Добавляем несколько переводов
	transfers := []storages.LargeTransfer{
		{UserID: 1, Amount: 50000.0},
		{UserID: 2, Amount: 60000.0},
		{UserID: 1, Amount: 70000.0},
		{UserID: 1, Amount: 80000.0},
	}
	storage.SaveTransferBatch(ctx, transfers)

	// Получаем переводы для пользователя 1
	userTransfers, err := storage.GetTransfersByUser(ctx, 1, 10)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(userTransfers) != 3 {
		t.Fatalf("Expected 3 transfers for user 1, got %d", len(userTransfers))
	}
}

func TestGetStatistics(t *testing.T) {
	storage := NewMockStorage()
	ctx := context.Background()

	// Добавляем переводы
	transfers := []storages.LargeTransfer{
		{UserID: 1, Amount: 50000.0, ProcessedAt: time.Now()},
		{UserID: 2, Amount: 60000.0, ProcessedAt: time.Now()},
		{UserID: 3, Amount: 70000.0, ProcessedAt: time.Now()},
	}
	storage.SaveTransferBatch(ctx, transfers)

	stats, err := storage.GetStatistics(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if stats.TotalProcessed != 3 {
		t.Fatalf("Expected 3 processed transfers, got %d", stats.TotalProcessed)
	}

	expectedAvg := (50000.0 + 60000.0 + 70000.0) / 3
	if stats.AverageAmount != expectedAvg {
		t.Fatalf("Expected average %.2f, got %.2f", expectedAvg, stats.AverageAmount)
	}

	expectedTotal := 180000.0
	if stats.TotalAmount != expectedTotal {
		t.Fatalf("Expected total %.2f, got %.2f", expectedTotal, stats.TotalAmount)
	}
}

func TestKafkaMessageParsing(t *testing.T) {
	// Тест парсинга JSON сообщения из Kafka
	jsonMsg := `{
		"user_id": 123,
		"type": "exchange",
		"from_currency": "USD",
		"to_currency": "EUR",
		"amount": 75000.50,
		"timestamp": "2024-02-02T15:04:05Z"
	}`

	var msg storages.KafkaMessage
	err := json.Unmarshal([]byte(jsonMsg), &msg)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if msg.UserID != 123 {
		t.Fatalf("Expected UserID 123, got %d", msg.UserID)
	}

	if msg.Type != "exchange" {
		t.Fatalf("Expected type 'exchange', got '%s'", msg.Type)
	}

	if msg.Amount != 75000.50 {
		t.Fatalf("Expected amount 75000.50, got %.2f", msg.Amount)
	}
}

func TestTransferValidation(t *testing.T) {
	transfer := &storages.LargeTransfer{
		UserID: 1,
		Type:   storages.TransferTypeDeposit,
		Amount: 50000.0,
	}

	// Проверка типа перевода
	validTypes := map[string]bool{
		storages.TransferTypeDeposit:  true,
		storages.TransferTypeWithdraw: true,
		storages.TransferTypeExchange: true,
	}

	if !validTypes[transfer.Type] {
		t.Fatalf("Invalid transfer type: %s", transfer.Type)
	}

	// Проверка суммы
	if transfer.Amount <= 0 {
		t.Fatal("Transfer amount must be positive")
	}
}
