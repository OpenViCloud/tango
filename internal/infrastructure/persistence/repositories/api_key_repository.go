package repositories

import (
	"context"
	"fmt"
	"time"

	"tango/internal/domain"
	"tango/internal/infrastructure/persistence/models"

	"gorm.io/gorm"
)

type APIKeyRepository struct {
	db   *gorm.DB
	base *BaseRepository[models.APIKeyRecord]
}

func NewAPIKeyRepository(db *gorm.DB) *APIKeyRepository {
	return &APIKeyRepository{
		db:   db,
		base: NewBaseRepository[models.APIKeyRecord](db),
	}
}

func (r *APIKeyRepository) Save(ctx context.Context, key *domain.APIKey) (*domain.APIKey, error) {
	record := toAPIKeyRecord(key)
	if err := r.base.Create(ctx, &record); err != nil {
		return nil, fmt.Errorf("save api key: %w", err)
	}
	return r.GetByID(ctx, key.ID)
}

func (r *APIKeyRepository) FindByHash(ctx context.Context, keyHash string) (*domain.APIKey, error) {
	var record models.APIKeyRecord
	err := r.base.First(ctx, &record, func(db *gorm.DB) *gorm.DB {
		return db.Where("key_hash = ?", keyHash)
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return toAPIKeyDomain(record), nil
}

func (r *APIKeyRepository) GetByID(ctx context.Context, id string) (*domain.APIKey, error) {
	var record models.APIKeyRecord
	err := r.base.First(ctx, &record, func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", id)
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrAPIKeyNotFound
		}
		return nil, err
	}
	return toAPIKeyDomain(record), nil
}

func (r *APIKeyRepository) ListByUserID(ctx context.Context, userID string) ([]*domain.APIKey, error) {
	var records []models.APIKeyRecord
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	keys := make([]*domain.APIKey, 0, len(records))
	for _, rec := range records {
		keys = append(keys, toAPIKeyDomain(rec))
	}
	return keys, nil
}

func (r *APIKeyRepository) UpdateLastUsed(ctx context.Context, id string) error {
	now := time.Now().UTC()
	_, err := r.base.Updates(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", id)
	}, map[string]any{
		"last_used_at": now,
		"updated_at":   now,
	})
	return err
}

func (r *APIKeyRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.APIKeyRecord{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete api key: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrAPIKeyNotFound
	}
	return nil
}

func toAPIKeyRecord(key *domain.APIKey) models.APIKeyRecord {
	return models.APIKeyRecord{
		ID:         key.ID,
		Name:       key.Name,
		KeyHash:    key.KeyHash,
		UserID:     key.UserID,
		ExpiresAt:  key.ExpiresAt,
		LastUsedAt: key.LastUsedAt,
		CreatedAt:  key.CreatedAt,
		UpdatedAt:  key.UpdatedAt,
	}
}

func toAPIKeyDomain(rec models.APIKeyRecord) *domain.APIKey {
	return &domain.APIKey{
		ID:         rec.ID,
		Name:       rec.Name,
		KeyHash:    rec.KeyHash,
		UserID:     rec.UserID,
		ExpiresAt:  rec.ExpiresAt,
		LastUsedAt: rec.LastUsedAt,
		CreatedAt:  rec.CreatedAt,
		UpdatedAt:  rec.UpdatedAt,
	}
}

var _ domain.APIKeyRepository = (*APIKeyRepository)(nil)
