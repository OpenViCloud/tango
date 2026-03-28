package repositories

import (
	"context"
	"fmt"
	"time"

	"tango/internal/domain"
	"tango/internal/infrastructure/persistence/models"

	"gorm.io/gorm"
)

type SourceConnectionRepository struct {
	db   *gorm.DB
	base *BaseRepository[models.SourceConnectionRecord]
}

func NewSourceConnectionRepository(db *gorm.DB) *SourceConnectionRepository {
	return &SourceConnectionRepository{
		db:   db,
		base: NewBaseRepository[models.SourceConnectionRecord](db),
	}
}

func (r *SourceConnectionRepository) Save(ctx context.Context, connection *domain.SourceConnection) (*domain.SourceConnection, error) {
	record := models.SourceConnectionRecord{
		ID:                connection.ID,
		UserID:            connection.UserID,
		SourceProviderID:  connection.SourceProviderID,
		Provider:          string(connection.Provider),
		DisplayName:       connection.DisplayName,
		AccountIdentifier: connection.AccountIdentifier,
		ExternalID:        connection.ExternalID,
		MetadataJSON:      connection.MetadataJSON,
		Status:            string(connection.Status),
		ExpiresAt:         connection.ExpiresAt,
		LastUsedAt:        connection.LastUsedAt,
		CreatedAt:         connection.CreatedAt,
		UpdatedAt:         connection.UpdatedAt,
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing models.SourceConnectionRecord
		queryErr := tx.Where(
			"user_id = ? AND provider = ? AND external_id = ?",
			record.UserID,
			record.Provider,
			record.ExternalID,
		).First(&existing).Error
		if queryErr != nil && queryErr != gorm.ErrRecordNotFound {
			return fmt.Errorf("find source connection: %w", queryErr)
		}

		if queryErr == gorm.ErrRecordNotFound {
			return tx.Create(&record).Error
		}

		record.ID = existing.ID
		record.CreatedAt = existing.CreatedAt
		return tx.Model(&models.SourceConnectionRecord{}).
			Where("id = ?", existing.ID).
			Updates(map[string]any{
				"display_name":       record.DisplayName,
				"source_provider_id": record.SourceProviderID,
				"metadata_json":      record.MetadataJSON,
				"status":             record.Status,
				"expires_at":         record.ExpiresAt,
				"last_used_at":       record.LastUsedAt,
				"updated_at":         record.UpdatedAt,
				"account_identifier": record.AccountIdentifier,
				"external_id":        record.ExternalID,
			}).Error
	})
	if err != nil {
		return nil, fmt.Errorf("save source connection: %w", err)
	}

	return r.GetByProviderAndAccount(ctx, connection.UserID, connection.Provider, connection.ExternalID)
}

func (r *SourceConnectionRepository) GetByID(ctx context.Context, id string) (*domain.SourceConnection, error) {
	var record models.SourceConnectionRecord
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrSourceConnectionNotFound
		}
		return nil, fmt.Errorf("get source connection by id: %w", err)
	}
	return toDomainSourceConnection(record), nil
}

func (r *SourceConnectionRepository) GetByProviderAndAccount(ctx context.Context, userID string, provider domain.SourceConnectionProvider, accountIdentifier string) (*domain.SourceConnection, error) {
	var record models.SourceConnectionRecord
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND provider = ? AND external_id = ?", userID, string(provider), accountIdentifier).
		First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrSourceConnectionNotFound
		}
		return nil, fmt.Errorf("get source connection by provider/account: %w", err)
	}
	return toDomainSourceConnection(record), nil
}

func (r *SourceConnectionRepository) ListByUser(ctx context.Context, userID string) ([]*domain.SourceConnection, error) {
	var records []models.SourceConnectionRecord
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list source connections: %w", err)
	}
	items := make([]*domain.SourceConnection, 0, len(records))
	for _, record := range records {
		items = append(items, toDomainSourceConnection(record))
	}
	return items, nil
}

func (r *SourceConnectionRepository) TouchUsedAt(ctx context.Context, id string, usedAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&models.SourceConnectionRecord{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"last_used_at": usedAt,
			"updated_at":   usedAt,
		})
	if result.Error != nil {
		return fmt.Errorf("touch source connection used at: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrSourceConnectionNotFound
	}
	return nil
}

func toDomainSourceConnection(record models.SourceConnectionRecord) *domain.SourceConnection {
	return &domain.SourceConnection{
		ID:                record.ID,
		UserID:            record.UserID,
		SourceProviderID:  record.SourceProviderID,
		Provider:          domain.SourceConnectionProvider(record.Provider),
		DisplayName:       record.DisplayName,
		AccountIdentifier: record.AccountIdentifier,
		ExternalID:        record.ExternalID,
		MetadataJSON:      record.MetadataJSON,
		Status:            domain.SourceConnectionStatus(record.Status),
		ExpiresAt:         record.ExpiresAt,
		LastUsedAt:        record.LastUsedAt,
		CreatedAt:         record.CreatedAt,
		UpdatedAt:         record.UpdatedAt,
	}
}

var _ domain.SourceConnectionRepository = (*SourceConnectionRepository)(nil)
