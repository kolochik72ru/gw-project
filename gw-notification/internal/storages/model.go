package storages

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// LargeTransfer представляет крупный денежный перевод
type LargeTransfer struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID       int64              `bson:"user_id" json:"user_id"`
	Type         string             `bson:"type" json:"type"` // deposit, withdraw, exchange
	FromCurrency string             `bson:"from_currency,omitempty" json:"from_currency,omitempty"`
	ToCurrency   string             `bson:"to_currency,omitempty" json:"to_currency,omitempty"`
	Amount       float64            `bson:"amount" json:"amount"`
	Timestamp    time.Time          `bson:"timestamp" json:"timestamp"`
	ProcessedAt  time.Time          `bson:"processed_at" json:"processed_at"`
	Status       string             `bson:"status" json:"status"` // processed, failed
	ErrorMessage string             `bson:"error_message,omitempty" json:"error_message,omitempty"`
}

// TransferType определяет типы переводов
const (
	TransferTypeDeposit  = "deposit"
	TransferTypeWithdraw = "withdraw"
	TransferTypeExchange = "exchange"
)

// TransferStatus определяет статусы обработки
const (
	StatusProcessed = "processed"
	StatusFailed    = "failed"
)

// KafkaMessage представляет сообщение из Kafka
type KafkaMessage struct {
	UserID       int64     `json:"user_id"`
	Type         string    `json:"type"`
	FromCurrency string    `json:"from_currency"`
	ToCurrency   string    `json:"to_currency"`
	Amount       float64   `json:"amount"`
	Timestamp    time.Time `json:"timestamp"`
}

// Statistics представляет статистику обработки
type Statistics struct {
	TotalProcessed   int64     `bson:"total_processed" json:"total_processed"`
	TotalFailed      int64     `bson:"total_failed" json:"total_failed"`
	LastProcessedAt  time.Time `bson:"last_processed_at" json:"last_processed_at"`
	AverageAmount    float64   `bson:"average_amount" json:"average_amount"`
	TotalAmount      float64   `bson:"total_amount" json:"total_amount"`
	ProcessingRate   float64   `json:"processing_rate"` // messages per second
}
