package repositories

import (
	"context"
	"fmt"
	"time"

	"tango/internal/domain"
	"tango/internal/infrastructure/persistence/models"

	"gorm.io/gorm"
)

type EnvironmentRepository struct {
	db   *gorm.DB
	base *BaseRepository[models.EnvironmentRecord]
}

func NewEnvironmentRepository(db *gorm.DB) *EnvironmentRepository {
	return &EnvironmentRepository{
		db:   db,
		base: NewBaseRepository[models.EnvironmentRecord](db),
	}
}

func (r *EnvironmentRepository) Create(ctx context.Context, input domain.CreateEnvironmentInput) (*domain.Environment, error) {
	record := models.EnvironmentRecord{
		ID:        input.ID,
		Name:      input.Name,
		ProjectID: input.ProjectID,
		CreatedAt: time.Now().UTC(),
	}
	if err := r.base.Create(ctx, &record); err != nil {
		return nil, fmt.Errorf("create environment: %w", err)
	}
	return r.GetByID(ctx, input.ID)
}

func (r *EnvironmentRepository) ListByProject(ctx context.Context, projectID string) ([]*domain.Environment, error) {
	var records []models.EnvironmentRecord
	if err := r.db.WithContext(ctx).Where("project_id = ?", projectID).Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list environments by project: %w", err)
	}
	items := make([]*domain.Environment, 0, len(records))
	for _, rec := range records {
		items = append(items, toDomainEnvironment(rec))
	}
	return items, nil
}

func (r *EnvironmentRepository) GetByID(ctx context.Context, id string) (*domain.Environment, error) {
	var record models.EnvironmentRecord
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrEnvironmentNotFound
		}
		return nil, fmt.Errorf("get environment by id: %w", err)
	}
	return toDomainEnvironment(record), nil
}

func (r *EnvironmentRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.EnvironmentRecord{})
	if result.Error != nil {
		return fmt.Errorf("delete environment: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrEnvironmentNotFound
	}
	return nil
}

func toDomainEnvironment(record models.EnvironmentRecord) *domain.Environment {
	return &domain.Environment{
		ID:        record.ID,
		Name:      record.Name,
		ProjectID: record.ProjectID,
		CreatedAt: record.CreatedAt,
	}
}

var _ domain.EnvironmentRepository = (*EnvironmentRepository)(nil)
