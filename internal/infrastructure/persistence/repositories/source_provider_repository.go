package repositories

import (
	"context"
	"fmt"

	"tango/internal/domain"
	"tango/internal/infrastructure/persistence/models"

	"gorm.io/gorm"
)

type SourceProviderRepository struct {
	db *gorm.DB
}

func NewSourceProviderRepository(db *gorm.DB) *SourceProviderRepository {
	return &SourceProviderRepository{db: db}
}

func (r *SourceProviderRepository) Save(ctx context.Context, provider *domain.SourceProvider) (*domain.SourceProvider, error) {
	record := models.SourceProviderRecord{
		ID:                   provider.ID,
		UserID:               provider.UserID,
		Provider:             string(provider.Provider),
		DisplayName:          provider.DisplayName,
		EncryptedCredentials: provider.EncryptedCredentials,
		MetadataJSON:         provider.MetadataJSON,
		Status:               string(provider.Status),
		CreatedAt:            provider.CreatedAt,
		UpdatedAt:            provider.UpdatedAt,
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing models.SourceProviderRecord
		queryErr := tx.Where("user_id = ? AND provider = ?", record.UserID, record.Provider).First(&existing).Error
		if queryErr != nil && queryErr != gorm.ErrRecordNotFound {
			return fmt.Errorf("find source provider: %w", queryErr)
		}
		if queryErr == gorm.ErrRecordNotFound {
			return tx.Create(&record).Error
		}
		record.ID = existing.ID
		record.CreatedAt = existing.CreatedAt
		return tx.Model(&models.SourceProviderRecord{}).
			Where("id = ?", existing.ID).
			Updates(map[string]any{
				"display_name":          record.DisplayName,
				"encrypted_credentials": record.EncryptedCredentials,
				"metadata_json":         record.MetadataJSON,
				"status":                record.Status,
				"updated_at":            record.UpdatedAt,
			}).Error
	})
	if err != nil {
		return nil, fmt.Errorf("save source provider: %w", err)
	}
	return r.GetByUserAndProvider(ctx, provider.UserID, provider.Provider)
}

func (r *SourceProviderRepository) GetByID(ctx context.Context, id string) (*domain.SourceProvider, error) {
	var record models.SourceProviderRecord
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrSourceProviderNotFound
		}
		return nil, fmt.Errorf("get source provider by id: %w", err)
	}
	return toDomainSourceProvider(record), nil
}

func (r *SourceProviderRepository) GetByUserAndProvider(ctx context.Context, userID string, provider domain.SourceConnectionProvider) (*domain.SourceProvider, error) {
	var record models.SourceProviderRecord
	if err := r.db.WithContext(ctx).Where("user_id = ? AND provider = ?", userID, string(provider)).First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrSourceProviderNotFound
		}
		return nil, fmt.Errorf("get source provider by user/provider: %w", err)
	}
	return toDomainSourceProvider(record), nil
}

func toDomainSourceProvider(record models.SourceProviderRecord) *domain.SourceProvider {
	return &domain.SourceProvider{
		ID:                   record.ID,
		UserID:               record.UserID,
		Provider:             domain.SourceConnectionProvider(record.Provider),
		DisplayName:          record.DisplayName,
		EncryptedCredentials: record.EncryptedCredentials,
		MetadataJSON:         record.MetadataJSON,
		Status:               domain.SourceProviderStatus(record.Status),
		CreatedAt:            record.CreatedAt,
		UpdatedAt:            record.UpdatedAt,
	}
}

var _ domain.SourceProviderRepository = (*SourceProviderRepository)(nil)
