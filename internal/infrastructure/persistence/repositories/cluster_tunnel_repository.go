package repositories

import (
	"context"
	"fmt"
	"time"

	"tango/internal/domain"
	"tango/internal/infrastructure/persistence/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClusterTunnelRepository struct {
	db *gorm.DB
}

func NewClusterTunnelRepository(db *gorm.DB) *ClusterTunnelRepository {
	return &ClusterTunnelRepository{db: db}
}

func (r *ClusterTunnelRepository) Save(ctx context.Context, t *domain.ClusterTunnel) (*domain.ClusterTunnel, error) {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	t.CreatedAt = now
	t.UpdatedAt = now

	rec := toTunnelRecord(t)
	if err := r.db.WithContext(ctx).Create(&rec).Error; err != nil {
		return nil, fmt.Errorf("save cluster tunnel: %w", err)
	}
	for _, e := range t.Exposures {
		expRec := toExposureRecord(t.ID, &e)
		if err := r.db.WithContext(ctx).Create(&expRec).Error; err != nil {
			return nil, fmt.Errorf("save tunnel exposure: %w", err)
		}
	}
	return r.GetByID(ctx, t.ID)
}

func (r *ClusterTunnelRepository) Update(ctx context.Context, t *domain.ClusterTunnel) (*domain.ClusterTunnel, error) {
	t.UpdatedAt = time.Now().UTC()
	rec := toTunnelRecord(t)

	if err := r.db.WithContext(ctx).Save(&rec).Error; err != nil {
		return nil, fmt.Errorf("update cluster tunnel: %w", err)
	}

	// Replace exposures: delete old, insert new.
	if err := r.db.WithContext(ctx).
		Where("cluster_tunnel_id = ?", t.ID).
		Delete(&models.TunnelExposureRecord{}).Error; err != nil {
		return nil, fmt.Errorf("delete old exposures: %w", err)
	}
	for _, e := range t.Exposures {
		expRec := toExposureRecord(t.ID, &e)
		if err := r.db.WithContext(ctx).Create(&expRec).Error; err != nil {
			return nil, fmt.Errorf("save tunnel exposure: %w", err)
		}
	}
	return r.GetByID(ctx, t.ID)
}

func (r *ClusterTunnelRepository) GetByClusterID(ctx context.Context, clusterID string) (*domain.ClusterTunnel, error) {
	var rec models.ClusterTunnelRecord
	if err := r.db.WithContext(ctx).
		Preload("Exposures").
		Where("cluster_id = ?", clusterID).
		First(&rec).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrClusterTunnelNotFound
		}
		return nil, fmt.Errorf("get cluster tunnel by cluster id: %w", err)
	}
	return toDomainTunnel(rec), nil
}

func (r *ClusterTunnelRepository) GetByID(ctx context.Context, id string) (*domain.ClusterTunnel, error) {
	var rec models.ClusterTunnelRecord
	if err := r.db.WithContext(ctx).
		Preload("Exposures").
		Where("id = ?", id).
		First(&rec).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrClusterTunnelNotFound
		}
		return nil, fmt.Errorf("get cluster tunnel by id: %w", err)
	}
	return toDomainTunnel(rec), nil
}

func (r *ClusterTunnelRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("cluster_tunnel_id = ?", id).
			Delete(&models.TunnelExposureRecord{}).Error; err != nil {
			return fmt.Errorf("delete exposures: %w", err)
		}
		result := tx.Delete(&models.ClusterTunnelRecord{ID: id})
		if result.Error != nil {
			return fmt.Errorf("delete cluster tunnel: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return domain.ErrClusterTunnelNotFound
		}
		return nil
	})
}

// ── mapping helpers ───────────────────────────────────────────────────────────

func toTunnelRecord(t *domain.ClusterTunnel) models.ClusterTunnelRecord {
	return models.ClusterTunnelRecord{
		ID:                     t.ID,
		ClusterID:              t.ClusterID,
		CloudflareConnectionID: t.CloudflareConnectionID,
		TunnelID:               t.TunnelID,
		TokenEnc:               t.TokenEnc,
		Namespace:              t.Namespace,
		CreatedAt:              t.CreatedAt,
		UpdatedAt:              t.UpdatedAt,
	}
}

func toExposureRecord(tunnelID string, e *domain.TunnelExposure) models.TunnelExposureRecord {
	id := e.ID
	if id == "" {
		id = uuid.NewString()
	}
	createdAt := e.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	return models.TunnelExposureRecord{
		ID:              id,
		ClusterTunnelID: tunnelID,
		Hostname:        e.Hostname,
		ServiceURL:      e.ServiceURL,
		CreatedAt:       createdAt,
	}
}

func toDomainTunnel(rec models.ClusterTunnelRecord) *domain.ClusterTunnel {
	exposures := make([]domain.TunnelExposure, 0, len(rec.Exposures))
	for _, e := range rec.Exposures {
		exposures = append(exposures, domain.TunnelExposure{
			ID:         e.ID,
			Hostname:   e.Hostname,
			ServiceURL: e.ServiceURL,
			CreatedAt:  e.CreatedAt,
		})
	}
	return &domain.ClusterTunnel{
		ID:                     rec.ID,
		ClusterID:              rec.ClusterID,
		CloudflareConnectionID: rec.CloudflareConnectionID,
		TunnelID:               rec.TunnelID,
		TokenEnc:               rec.TokenEnc,
		Namespace:              rec.Namespace,
		Exposures:              exposures,
		CreatedAt:              rec.CreatedAt,
		UpdatedAt:              rec.UpdatedAt,
	}
}

var _ domain.ClusterTunnelRepository = (*ClusterTunnelRepository)(nil)
