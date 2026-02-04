package mongodb

import (
	"context"
	"fmt"
	"time"

	"gw-notification/internal/storages"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SaveTransfer сохраняет информацию о крупном переводе
func (s *MongoStorage) SaveTransfer(ctx context.Context, transfer *storages.LargeTransfer) error {
	transfer.ProcessedAt = time.Now()
	transfer.Status = storages.StatusProcessed

	result, err := s.collection.InsertOne(ctx, transfer)
	if err != nil {
		s.logger.Errorf("Failed to save transfer: %v", err)
		return fmt.Errorf("failed to save transfer: %w", err)
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		transfer.ID = oid
	}

	s.logger.Debugf("Saved transfer: UserID=%d, Amount=%.2f, Type=%s",
		transfer.UserID, transfer.Amount, transfer.Type)

	return nil
}

// SaveTransferBatch сохраняет пакет переводов
func (s *MongoStorage) SaveTransferBatch(ctx context.Context, transfers []storages.LargeTransfer) error {
	if len(transfers) == 0 {
		return nil
	}

	// Подготовка документов для вставки
	documents := make([]interface{}, len(transfers))
	now := time.Now()

	for i := range transfers {
		transfers[i].ProcessedAt = now
		transfers[i].Status = storages.StatusProcessed
		documents[i] = transfers[i]
	}

	// Вставка пакетом
	result, err := s.collection.InsertMany(ctx, documents)
	if err != nil {
		s.logger.Errorf("Failed to save transfer batch: %v", err)
		return fmt.Errorf("failed to save transfer batch: %w", err)
	}

	s.logger.Infof("Saved batch of %d transfers (inserted: %d)",
		len(transfers), len(result.InsertedIDs))

	return nil
}

// GetTransfer получает перевод по ID
func (s *MongoStorage) GetTransfer(ctx context.Context, id string) (*storages.LargeTransfer, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid ID format: %w", err)
	}

	filter := bson.M{"_id": objectID}

	var transfer storages.LargeTransfer
	err = s.collection.FindOne(ctx, filter).Decode(&transfer)
	if err != nil {
		s.logger.Errorf("Failed to get transfer: %v", err)
		return nil, fmt.Errorf("failed to get transfer: %w", err)
	}

	return &transfer, nil
}

// GetTransfersByUser получает переводы пользователя
func (s *MongoStorage) GetTransfersByUser(ctx context.Context, userID int64, limit int) ([]storages.LargeTransfer, error) {
	filter := bson.M{"user_id": userID}
	opts := options.Find().
		SetSort(bson.D{{Key: "timestamp", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := s.collection.Find(ctx, filter, opts)
	if err != nil {
		s.logger.Errorf("Failed to query transfers: %v", err)
		return nil, fmt.Errorf("failed to query transfers: %w", err)
	}
	defer cursor.Close(ctx)

	var transfers []storages.LargeTransfer
	if err := cursor.All(ctx, &transfers); err != nil {
		s.logger.Errorf("Failed to decode transfers: %v", err)
		return nil, fmt.Errorf("failed to decode transfers: %w", err)
	}

	s.logger.Debugf("Retrieved %d transfers for user %d", len(transfers), userID)
	return transfers, nil
}

// GetRecentTransfers получает последние переводы
func (s *MongoStorage) GetRecentTransfers(ctx context.Context, limit int) ([]storages.LargeTransfer, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "processed_at", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := s.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		s.logger.Errorf("Failed to query recent transfers: %v", err)
		return nil, fmt.Errorf("failed to query recent transfers: %w", err)
	}
	defer cursor.Close(ctx)

	var transfers []storages.LargeTransfer
	if err := cursor.All(ctx, &transfers); err != nil {
		s.logger.Errorf("Failed to decode transfers: %v", err)
		return nil, fmt.Errorf("failed to decode transfers: %w", err)
	}

	s.logger.Debugf("Retrieved %d recent transfers", len(transfers))
	return transfers, nil
}

// GetStatistics возвращает статистику обработки
func (s *MongoStorage) GetStatistics(ctx context.Context) (*storages.Statistics, error) {
	pipeline := []bson.M{
		{
			"$group": bson.M{
				"_id": nil,
				"total_processed": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{
							bson.M{"$eq": []string{"$status", storages.StatusProcessed}},
							1,
							0,
						},
					},
				},
				"total_failed": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{
							bson.M{"$eq": []string{"$status", storages.StatusFailed}},
							1,
							0,
						},
					},
				},
				"average_amount": bson.M{"$avg": "$amount"},
				"total_amount":   bson.M{"$sum": "$amount"},
				"last_processed": bson.M{"$max": "$processed_at"},
			},
		},
	}

	cursor, err := s.collection.Aggregate(ctx, pipeline)
	if err != nil {
		s.logger.Errorf("Failed to get statistics: %v", err)
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}
	defer cursor.Close(ctx)

	var results []struct {
		TotalProcessed  int64     `bson:"total_processed"`
		TotalFailed     int64     `bson:"total_failed"`
		AverageAmount   float64   `bson:"average_amount"`
		TotalAmount     float64   `bson:"total_amount"`
		LastProcessedAt time.Time `bson:"last_processed"`
	}

	if err := cursor.All(ctx, &results); err != nil {
		s.logger.Errorf("Failed to decode statistics: %v", err)
		return nil, fmt.Errorf("failed to decode statistics: %w", err)
	}

	stats := &storages.Statistics{}
	if len(results) > 0 {
		stats.TotalProcessed = results[0].TotalProcessed
		stats.TotalFailed = results[0].TotalFailed
		stats.AverageAmount = results[0].AverageAmount
		stats.TotalAmount = results[0].TotalAmount
		stats.LastProcessedAt = results[0].LastProcessedAt
	}

	s.logger.Debugf("Statistics: Processed=%d, Failed=%d, Avg=%.2f",
		stats.TotalProcessed, stats.TotalFailed, stats.AverageAmount)

	return stats, nil
}
