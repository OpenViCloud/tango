package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"tango/internal/domain"
	"tango/internal/infrastructure/persistence/models"

	"gorm.io/gorm"
)

type ResourceRunRepository struct {
	db *gorm.DB
}

func NewResourceRunRepository(db *gorm.DB) *ResourceRunRepository {
	return &ResourceRunRepository{db: db}
}

func (r *ResourceRunRepository) Save(ctx context.Context, run *domain.ResourceRun) (*domain.ResourceRun, error) {
	now := time.Now().UTC()
	run.CreatedAt = now
	run.UpdatedAt = now

	record := toResourceRunRecord(run)
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, fmt.Errorf("save resource run: %w", err)
	}
	return r.GetByID(ctx, run.ID)
}

func (r *ResourceRunRepository) Update(ctx context.Context, run *domain.ResourceRun) (*domain.ResourceRun, error) {
	run.UpdatedAt = time.Now().UTC()
	result := r.db.WithContext(ctx).
		Model(&models.ResourceRunRecord{}).
		Where("id = ?", run.ID).
		Updates(map[string]any{
			"status":      string(run.Status),
			"logs":        run.Logs,
			"error_msg":   run.ErrorMsg,
			"started_at":  run.StartedAt,
			"finished_at": run.FinishedAt,
			"updated_at":  run.UpdatedAt,
		})
	if result.Error != nil {
		return nil, fmt.Errorf("update resource run: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, domain.ErrResourceRunNotFound
	}
	return r.GetByID(ctx, run.ID)
}

func (r *ResourceRunRepository) GetByID(ctx context.Context, id string) (*domain.ResourceRun, error) {
	var record models.ResourceRunRecord
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrResourceRunNotFound
		}
		return nil, fmt.Errorf("get resource run: %w", err)
	}
	return toDomainResourceRun(&record), nil
}

func toResourceRunRecord(run *domain.ResourceRun) models.ResourceRunRecord {
	return models.ResourceRunRecord{
		ID:         run.ID,
		ResourceID: run.ResourceID,
		Status:     string(run.Status),
		Logs:       run.Logs,
		ErrorMsg:   run.ErrorMsg,
		StartedAt:  run.StartedAt,
		FinishedAt: run.FinishedAt,
		CreatedAt:  run.CreatedAt,
		UpdatedAt:  run.UpdatedAt,
	}
}

func toDomainResourceRun(record *models.ResourceRunRecord) *domain.ResourceRun {
	return &domain.ResourceRun{
		ID:         record.ID,
		ResourceID: record.ResourceID,
		Status:     domain.ResourceRunStatus(record.Status),
		Logs:       record.Logs,
		ErrorMsg:   record.ErrorMsg,
		StartedAt:  record.StartedAt,
		FinishedAt: record.FinishedAt,
		CreatedAt:  record.CreatedAt,
		UpdatedAt:  record.UpdatedAt,
	}
}

var _ domain.ResourceRunRepository = (*ResourceRunRepository)(nil)
