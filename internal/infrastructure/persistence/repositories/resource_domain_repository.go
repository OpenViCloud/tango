package repositories

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"tango/internal/domain"
	"tango/internal/infrastructure/persistence/models"

	"gorm.io/gorm"
)

func genID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

type resourceDomainRepository struct {
	db *gorm.DB
}

func NewResourceDomainRepository(db *gorm.DB) domain.ResourceDomainRepository {
	return &resourceDomainRepository{db: db}
}

func (r *resourceDomainRepository) Create(ctx context.Context, d domain.ResourceDomain) (*domain.ResourceDomain, error) {
	id := d.ID
	if id == "" {
		id = genID()
	}
	rec := models.ResourceDomainRecord{
		ID:         id,
		ResourceID: d.ResourceID,
		Host:       d.Host,
		TLSEnabled: d.TLSEnabled,
		Type:       d.Type,
		Verified:   d.Verified,
		VerifiedAt: d.VerifiedAt,
	}
	if err := r.db.WithContext(ctx).Create(&rec).Error; err != nil {
		if isUniqueConstraintError(err) {
			return nil, domain.ErrResourceDomainConflict
		}
		return nil, err
	}
	return r.toEntity(&rec), nil
}

func (r *resourceDomainRepository) ListByResource(ctx context.Context, resourceID string) ([]*domain.ResourceDomain, error) {
	var records []models.ResourceDomainRecord
	if err := r.db.WithContext(ctx).
		Where("resource_id = ?", resourceID).
		Order("created_at").
		Find(&records).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.ResourceDomain, 0, len(records))
	for i := range records {
		out = append(out, r.toEntity(&records[i]))
	}
	return out, nil
}

func (r *resourceDomainRepository) GetByID(ctx context.Context, id string) (*domain.ResourceDomain, error) {
	var rec models.ResourceDomainRecord
	if err := r.db.WithContext(ctx).First(&rec, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrResourceDomainNotFound
		}
		return nil, err
	}
	return r.toEntity(&rec), nil
}

func (r *resourceDomainRepository) GetByHost(ctx context.Context, host string) (*domain.ResourceDomain, error) {
	var rec models.ResourceDomainRecord
	if err := r.db.WithContext(ctx).First(&rec, "host = ?", host).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrResourceDomainNotFound
		}
		return nil, err
	}
	return r.toEntity(&rec), nil
}

func (r *resourceDomainRepository) Update(ctx context.Context, d domain.ResourceDomain) (*domain.ResourceDomain, error) {
	updates := map[string]any{
		"host":        d.Host,
		"tls_enabled": d.TLSEnabled,
		"verified":    d.Verified,
		"verified_at": d.VerifiedAt,
		"updated_at":  time.Now(),
	}
	if err := r.db.WithContext(ctx).
		Model(&models.ResourceDomainRecord{}).
		Where("id = ?", d.ID).
		Updates(updates).Error; err != nil {
		if isUniqueConstraintError(err) {
			return nil, domain.ErrResourceDomainConflict
		}
		return nil, err
	}
	return r.GetByID(ctx, d.ID)
}

func (r *resourceDomainRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&models.ResourceDomainRecord{}, "id = ?", id).Error
}

func (r *resourceDomainRepository) DeleteByResource(ctx context.Context, resourceID string) error {
	return r.db.WithContext(ctx).Delete(&models.ResourceDomainRecord{}, "resource_id = ?", resourceID).Error
}

func (r *resourceDomainRepository) SetVerified(ctx context.Context, id string, verifiedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&models.ResourceDomainRecord{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"verified":    true,
			"verified_at": verifiedAt,
			"updated_at":  time.Now(),
		}).Error
}

func (r *resourceDomainRepository) toEntity(rec *models.ResourceDomainRecord) *domain.ResourceDomain {
	return &domain.ResourceDomain{
		ID:         rec.ID,
		ResourceID: rec.ResourceID,
		Host:       rec.Host,
		TLSEnabled: rec.TLSEnabled,
		Type:       rec.Type,
		Verified:   rec.Verified,
		VerifiedAt: rec.VerifiedAt,
		CreatedAt:  rec.CreatedAt,
		UpdatedAt:  rec.UpdatedAt,
	}
}
