package repositories

import (
	"context"
	"errors"
	"time"

	"tango/internal/domain"
	"tango/internal/infrastructure/persistence/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type platformConfigRepository struct {
	db *gorm.DB
}

func NewPlatformConfigRepository(db *gorm.DB) domain.PlatformConfigRepository {
	return &platformConfigRepository{db: db}
}

func (r *platformConfigRepository) Get(ctx context.Context, key string) (*domain.PlatformConfig, error) {
	var rec models.PlatformConfigRecord
	if err := r.db.WithContext(ctx).First(&rec, "key = ?", key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrPlatformConfigNotFound
		}
		return nil, err
	}
	return &domain.PlatformConfig{
		Key:       rec.Key,
		Value:     rec.Value,
		UpdatedAt: rec.UpdatedAt,
	}, nil
}

func (r *platformConfigRepository) Set(ctx context.Context, key, value string) error {
	rec := models.PlatformConfigRecord{
		Key:       key,
		Value:     value,
		UpdatedAt: time.Now(),
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).
		Create(&rec).Error
}

func (r *platformConfigRepository) List(ctx context.Context) ([]*domain.PlatformConfig, error) {
	var records []models.PlatformConfigRecord
	if err := r.db.WithContext(ctx).Order("key").Find(&records).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.PlatformConfig, 0, len(records))
	for _, rec := range records {
		out = append(out, &domain.PlatformConfig{
			Key:       rec.Key,
			Value:     rec.Value,
			UpdatedAt: rec.UpdatedAt,
		})
	}
	return out, nil
}
