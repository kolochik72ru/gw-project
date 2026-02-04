package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// LargeTransferMessage сообщение о крупном переводе
type LargeTransferMessage struct {
	UserID       int64     `json:"user_id"`
	Type         string    `json:"type"`
	FromCurrency string    `json:"from_currency"`
	ToCurrency   string    `json:"to_currency"`
	Amount       float64   `json:"amount"`
	Timestamp    time.Time `json:"timestamp"`
}

// Producer Kafka producer для отправки сообщений
type Producer struct {
	writer    *kafka.Writer
	threshold float64
	logger    *logrus.Logger
}

// NewProducer создает новый Kafka producer
func NewProducer(brokers []string, topic string, threshold float64, logger *logrus.Logger) *Producer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        true, // Асинхронная отправка для производительности
		Compression:  kafka.Snappy,
		BatchTimeout: 10 * time.Millisecond,
	}

	logger.Infof("Kafka producer initialized for topic: %s", topic)

	return &Producer{
		writer:    writer,
		threshold: threshold,
		logger:    logger,
	}
}

// SendLargeTransferNotification отправляет уведомление о крупном переводе, если сумма превышает порог
func (p *Producer) SendLargeTransferNotification(ctx context.Context, userID int64, transferType, fromCurrency, toCurrency string, amount float64) error {
	// Проверяем, превышает ли сумма порог
	if amount < p.threshold {
		p.logger.Debugf("Transfer amount %.2f is below threshold %.2f, skipping Kafka notification", amount, p.threshold)
		return nil
	}

	message := LargeTransferMessage{
		UserID:       userID,
		Type:         transferType,
		FromCurrency: fromCurrency,
		ToCurrency:   toCurrency,
		Amount:       amount,
		Timestamp:    time.Now(),
	}

	// Сериализуем сообщение в JSON
	messageBytes, err := json.Marshal(message)
	if err != nil {
		p.logger.Errorf("Failed to marshal Kafka message: %v", err)
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Отправляем сообщение в Kafka
	kafkaMessage := kafka.Message{
		Key:   []byte(fmt.Sprintf("user_%d", userID)),
		Value: messageBytes,
		Time:  time.Now(),
	}

	err = p.writer.WriteMessages(ctx, kafkaMessage)
	if err != nil {
		p.logger.Errorf("Failed to send message to Kafka: %v", err)
		return fmt.Errorf("failed to send message: %w", err)
	}

	p.logger.Infof("Sent large transfer notification to Kafka: UserID=%d, Amount=%.2f %s",
		userID, amount, fromCurrency)

	return nil
}

// Close закрывает Kafka producer
func (p *Producer) Close() error {
	if p.writer != nil {
		p.logger.Info("Closing Kafka producer")
		return p.writer.Close()
	}
	return nil
}
