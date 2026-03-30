package repositories

import (
	"context"
	"errors"
	"time"

	"tango/internal/domain"
	"tango/internal/infrastructure/persistence/models"

	"gorm.io/gorm"
)

type baseDomainRepository struct {
	db *gorm.DB
}

func NewBaseDomainRepository(db *gorm.DB) domain.BaseDomainRepository {
	return &baseDomainRepository{db: db}
}

func (r *baseDomainRepository) Create(ctx context.Context, bd domain.BaseDomain) (*domain.BaseDomain, error) {
	id := bd.ID
	if id == "" {
		id = genID()
	}
	now := time.Now().UTC()
	rec := models.BaseDomainRecord{
		ID:              id,
		Domain:          bd.Domain,
		WildcardEnabled: bd.WildcardEnabled,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := r.db.WithContext(ctx).Create(&rec).Error; err != nil {
		if isUniqueConstraintError(err) {
			return nil, domain.ErrBaseDomainConflict
		}
		return nil, err
	}
	return r.toEntity(&rec), nil
}

func (r *baseDomainRepository) List(ctx context.Context) ([]*domain.BaseDomain, error) {
	var records []models.BaseDomainRecord
	if err := r.db.WithContext(ctx).Order("created_at").Find(&records).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.BaseDomain, 0, len(records))
	for i := range records {
		out = append(out, r.toEntity(&records[i]))
	}
	return out, nil
}

func (r *baseDomainRepository) GetByID(ctx context.Context, id string) (*domain.BaseDomain, error) {
	var rec models.BaseDomainRecord
	if err := r.db.WithContext(ctx).First(&rec, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrBaseDomainNotFound
		}
		return nil, err
	}
	return r.toEntity(&rec), nil
}

func (r *baseDomainRepository) GetByDomain(ctx context.Context, d string) (*domain.BaseDomain, error) {
	var rec models.BaseDomainRecord
	if err := r.db.WithContext(ctx).First(&rec, "domain = ?", d).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrBaseDomainNotFound
		}
		return nil, err
	}
	return r.toEntity(&rec), nil
}

func (r *baseDomainRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&models.BaseDomainRecord{}, "id = ?", id).Error
}

func (r *baseDomainRepository) toEntity(rec *models.BaseDomainRecord) *domain.BaseDomain {
	return &domain.BaseDomain{
		ID:              rec.ID,
		Domain:          rec.Domain,
		WildcardEnabled: rec.WildcardEnabled,
		CreatedAt:       rec.CreatedAt,
		UpdatedAt:       rec.UpdatedAt,
	}
}
