package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"gw-notification/internal/storages"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// Consumer Kafka consumer для получения сообщений
type Consumer struct {
	reader        *kafka.Reader
	storage       storages.Storage
	logger        *logrus.Logger
	batchSize     int
	workers       int
	flushInterval time.Duration
	retryAttempts int
	retryDelay    time.Duration

	// Статистика
	mu                sync.RWMutex
	messagesProcessed int64
	messagesFailed    int64
	startTime         time.Time
}

// Config конфигурация consumer
type Config struct {
	Brokers       []string
	Topic         string
	GroupID       string
	Partition     int
	MinBytes      int
	MaxBytes      int
	MaxWait       time.Duration
	BatchSize     int
	Workers       int
	FlushInterval time.Duration
	RetryAttempts int
	RetryDelay    time.Duration
}

// NewConsumer создает новый Kafka consumer
func NewConsumer(cfg *Config, storage storages.Storage, logger *logrus.Logger) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   cfg.Brokers,
		Topic:     cfg.Topic,
		GroupID:   cfg.GroupID,
		Partition: cfg.Partition,
		MinBytes:  cfg.MinBytes,
		MaxBytes:  cfg.MaxBytes,
		MaxWait:   cfg.MaxWait,
		Logger:    kafka.LoggerFunc(logger.Debugf),
		ErrorLogger: kafka.LoggerFunc(logger.Errorf),
	})

	logger.Infof("Kafka consumer initialized: Topic=%s, GroupID=%s, Brokers=%v",
		cfg.Topic, cfg.GroupID, cfg.Brokers)

	return &Consumer{
		reader:        reader,
		storage:       storage,
		logger:        logger,
		batchSize:     cfg.BatchSize,
		workers:       cfg.Workers,
		flushInterval: cfg.FlushInterval,
		retryAttempts: cfg.RetryAttempts,
		retryDelay:    cfg.RetryDelay,
		startTime:     time.Now(),
	}
}

// Start запускает consumer
func (c *Consumer) Start(ctx context.Context) error {
	c.logger.Info("Starting Kafka consumer...")

	// Создаем канал для сообщений
	messages := make(chan kafka.Message, c.batchSize*2)

	// Запускаем воркеры для обработки
	var wg sync.WaitGroup
	for i := 0; i < c.workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			c.processMessages(ctx, messages, workerID)
		}(i)
	}

	// Запускаем чтение сообщений
	go func() {
		defer close(messages)
		c.readMessages(ctx, messages)
	}()

	// Ждем завершения всех воркеров
	wg.Wait()

	c.logger.Info("Kafka consumer stopped")
	return nil
}

// readMessages читает сообщения из Kafka
func (c *Consumer) readMessages(ctx context.Context, messages chan<- kafka.Message) {
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Stopping message reading...")
			return
		default:
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				c.logger.Errorf("Failed to fetch message: %v", err)
				time.Sleep(c.retryDelay)
				continue
			}

			messages <- msg
		}
	}
}

// processMessages обрабатывает сообщения из канала
func (c *Consumer) processMessages(ctx context.Context, messages <-chan kafka.Message, workerID int) {
	batch := make([]storages.LargeTransfer, 0, c.batchSize)
	kafkaMessages := make([]kafka.Message, 0, c.batchSize)

	ticker := time.NewTicker(c.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Сохраняем оставшиеся сообщения перед выходом
			if len(batch) > 0 {
				c.flushBatch(ctx, batch, kafkaMessages)
			}
			return

		case <-ticker.C:
			// Периодическое сохранение пакета
			if len(batch) > 0 {
				c.flushBatch(ctx, batch, kafkaMessages)
				batch = batch[:0]
				kafkaMessages = kafkaMessages[:0]
			}

		case msg, ok := <-messages:
			if !ok {
				// Канал закрыт, сохраняем оставшееся
				if len(batch) > 0 {
					c.flushBatch(ctx, batch, kafkaMessages)
				}
				return
			}

			// Парсим сообщение
			transfer, err := c.parseMessage(msg)
			if err != nil {
				c.logger.Errorf("Worker %d: Failed to parse message: %v", workerID, err)
				c.incrementFailed()
				// Все равно коммитим, чтобы не блокировать очередь
				if err := c.reader.CommitMessages(ctx, msg); err != nil {
					c.logger.Errorf("Worker %d: Failed to commit failed message: %v", workerID, err)
				}
				continue
			}

			// Добавляем в пакет
			batch = append(batch, *transfer)
			kafkaMessages = append(kafkaMessages, msg)

			// Если пакет заполнен, сохраняем
			if len(batch) >= c.batchSize {
				c.flushBatch(ctx, batch, kafkaMessages)
				batch = batch[:0]
				kafkaMessages = kafkaMessages[:0]
			}
		}
	}
}

// parseMessage парсит сообщение из Kafka
func (c *Consumer) parseMessage(msg kafka.Message) (*storages.LargeTransfer, error) {
	var kafkaMsg storages.KafkaMessage
	if err := json.Unmarshal(msg.Value, &kafkaMsg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	transfer := &storages.LargeTransfer{
		UserID:       kafkaMsg.UserID,
		Type:         kafkaMsg.Type,
		FromCurrency: kafkaMsg.FromCurrency,
		ToCurrency:   kafkaMsg.ToCurrency,
		Amount:       kafkaMsg.Amount,
		Timestamp:    kafkaMsg.Timestamp,
	}

	return transfer, nil
}

// flushBatch сохраняет пакет сообщений в MongoDB
func (c *Consumer) flushBatch(ctx context.Context, batch []storages.LargeTransfer, messages []kafka.Message) {
	if len(batch) == 0 {
		return
	}

	start := time.Now()

	// Пытаемся сохранить пакет с повторами
	var err error
	for attempt := 0; attempt < c.retryAttempts; attempt++ {
		err = c.storage.SaveTransferBatch(ctx, batch)
		if err == nil {
			break
		}

		c.logger.Warnf("Attempt %d/%d: Failed to save batch: %v",
			attempt+1, c.retryAttempts, err)

		if attempt < c.retryAttempts-1 {
			time.Sleep(c.retryDelay)
		}
	}

	if err != nil {
		c.logger.Errorf("Failed to save batch after %d attempts: %v", c.retryAttempts, err)
		c.incrementFailed()
		return
	}

	// Коммитим сообщения в Kafka
	if err := c.reader.CommitMessages(ctx, messages...); err != nil {
		c.logger.Errorf("Failed to commit messages: %v", err)
		return
	}

	duration := time.Since(start)
	c.incrementProcessed(int64(len(batch)))

	c.logger.Infof("Flushed batch: size=%d, duration=%v, rate=%.2f msg/s",
		len(batch), duration, float64(len(batch))/duration.Seconds())
}

// incrementProcessed увеличивает счетчик обработанных сообщений
func (c *Consumer) incrementProcessed(count int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messagesProcessed += count
}

// incrementFailed увеличивает счетчик неудачных сообщений
func (c *Consumer) incrementFailed() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messagesFailed++
}

// GetStatistics возвращает статистику обработки
func (c *Consumer) GetStatistics() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	duration := time.Since(c.startTime)
	rate := float64(c.messagesProcessed) / duration.Seconds()

	return map[string]interface{}{
		"messages_processed": c.messagesProcessed,
		"messages_failed":    c.messagesFailed,
		"processing_rate":    rate,
		"uptime_seconds":     duration.Seconds(),
	}
}

// Close закрывает consumer
func (c *Consumer) Close() error {
	c.logger.Info("Closing Kafka consumer")
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}
