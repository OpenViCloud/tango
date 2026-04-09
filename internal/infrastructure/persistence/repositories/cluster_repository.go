package repositories

import (
	"context"
	"fmt"
	"time"

	"tango/internal/domain"
	"tango/internal/infrastructure/persistence/models"

	"gorm.io/gorm"
)

type ClusterRepository struct {
	db   *gorm.DB
	base *BaseRepository[models.ClusterRecord]
}

func NewClusterRepository(db *gorm.DB) *ClusterRepository {
	return &ClusterRepository{
		db:   db,
		base: NewBaseRepository[models.ClusterRecord](db),
	}
}

func (r *ClusterRepository) Save(ctx context.Context, c *domain.Cluster) (*domain.Cluster, error) {
	record := toClusterRecord(c)
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, fmt.Errorf("save cluster: %w", err)
	}
	for _, n := range c.Nodes {
		node := models.ClusterNodeRecord{
			ClusterID: c.ID,
			ServerID:  n.ServerID,
			Role:      string(n.Role),
		}
		if err := r.db.WithContext(ctx).Create(&node).Error; err != nil {
			return nil, fmt.Errorf("save cluster node: %w", err)
		}
	}
	return r.GetByID(ctx, c.ID)
}

func (r *ClusterRepository) Update(ctx context.Context, c *domain.Cluster) (*domain.Cluster, error) {
	record := toClusterRecord(c)
	record.UpdatedAt = time.Now().UTC()
	rowsAffected, err := r.base.Updates(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", c.ID)
	}, record)
	if err != nil {
		return nil, fmt.Errorf("update cluster: %w", err)
	}
	if rowsAffected == 0 {
		return nil, domain.ErrClusterNotFound
	}
	return r.GetByID(ctx, c.ID)
}

func (r *ClusterRepository) UpdateStatus(ctx context.Context, id string, status domain.ClusterStatus, errMsg string) error {
	result := r.db.WithContext(ctx).Model(&models.ClusterRecord{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     string(status),
			"error_msg":  errMsg,
			"updated_at": time.Now().UTC(),
		})
	if result.Error != nil {
		return fmt.Errorf("update cluster status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrClusterNotFound
	}
	return nil
}

func (r *ClusterRepository) UpdateKubeconfig(ctx context.Context, id string, kubeconfigEnc string) error {
	result := r.db.WithContext(ctx).Model(&models.ClusterRecord{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"kubeconfig_enc": kubeconfigEnc,
			"updated_at":     time.Now().UTC(),
		})
	if result.Error != nil {
		return fmt.Errorf("update cluster kubeconfig: %w", result.Error)
	}
	return nil
}

func (r *ClusterRepository) GetByID(ctx context.Context, id string) (*domain.Cluster, error) {
	var record models.ClusterRecord
	if err := r.db.WithContext(ctx).
		Preload("Nodes").
		Where("id = ?", id).
		First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrClusterNotFound
		}
		return nil, err
	}
	return toDomainCluster(record), nil
}

func (r *ClusterRepository) ListAll(ctx context.Context) ([]*domain.Cluster, error) {
	var records []models.ClusterRecord
	if err := r.db.WithContext(ctx).
		Preload("Nodes").
		Order("created_at DESC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list clusters: %w", err)
	}
	items := make([]*domain.Cluster, 0, len(records))
	for _, rec := range records {
		items = append(items, toDomainCluster(rec))
	}
	return items, nil
}

func (r *ClusterRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete child nodes first to avoid FK constraint violation
		if err := tx.Where("cluster_id = ?", id).Delete(&models.ClusterNodeRecord{}).Error; err != nil {
			return fmt.Errorf("delete cluster nodes: %w", err)
		}
		result := tx.Delete(&models.ClusterRecord{ID: id})
		if result.Error != nil {
			return fmt.Errorf("delete cluster: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return domain.ErrClusterNotFound
		}
		return nil
	})
}

func toClusterRecord(c *domain.Cluster) models.ClusterRecord {
	return models.ClusterRecord{
		ID:            c.ID,
		Name:          c.Name,
		Status:        string(c.Status),
		ErrorMsg:      c.ErrorMsg,
		K8sVersion:    c.K8sVersion,
		PodCIDR:       c.PodCIDR,
		KubeconfigEnc: c.Kubeconfig,
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
	}
}

func toDomainCluster(r models.ClusterRecord) *domain.Cluster {
	nodes := make([]domain.ClusterNode, 0, len(r.Nodes))
	for _, n := range r.Nodes {
		nodes = append(nodes, domain.ClusterNode{
			ServerID: n.ServerID,
			Role:     domain.ClusterNodeRole(n.Role),
		})
	}
	return &domain.Cluster{
		ID:         r.ID,
		Name:       r.Name,
		Status:     domain.ClusterStatus(r.Status),
		ErrorMsg:   r.ErrorMsg,
		K8sVersion: r.K8sVersion,
		PodCIDR:    r.PodCIDR,
		Kubeconfig: r.KubeconfigEnc,
		Nodes:      nodes,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
}

var _ domain.ClusterRepository = (*ClusterRepository)(nil)
