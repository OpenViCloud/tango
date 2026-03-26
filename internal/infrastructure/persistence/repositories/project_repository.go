package repositories

import (
	"context"
	"fmt"
	"time"

	"tango/internal/domain"
	"tango/internal/infrastructure/persistence/models"

	"gorm.io/gorm"
)

type ProjectRepository struct {
	db   *gorm.DB
	base *BaseRepository[models.ProjectRecord]
}

func NewProjectRepository(db *gorm.DB) *ProjectRepository {
	return &ProjectRepository{
		db:   db,
		base: NewBaseRepository[models.ProjectRecord](db),
	}
}

func (r *ProjectRepository) Create(ctx context.Context, input domain.CreateProjectInput) (*domain.Project, error) {
	now := time.Now().UTC()
	record := models.ProjectRecord{
		ID:          input.ID,
		Name:        input.Name,
		Description: input.Description,
		CreatedBy:   input.CreatedBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := r.base.Create(ctx, &record); err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}
	return r.GetByID(ctx, input.ID)
}

func (r *ProjectRepository) List(ctx context.Context) ([]*domain.Project, error) {
	var records []models.ProjectRecord
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	items := make([]*domain.Project, 0, len(records))
	for _, rec := range records {
		items = append(items, toDomainProject(rec))
	}
	return items, nil
}

func (r *ProjectRepository) GetByID(ctx context.Context, id string) (*domain.Project, error) {
	var record models.ProjectRecord
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrProjectNotFound
		}
		return nil, fmt.Errorf("get project by id: %w", err)
	}
	return toDomainProject(record), nil
}

func (r *ProjectRepository) Update(ctx context.Context, id, name, description string) (*domain.Project, error) {
	now := time.Now().UTC()
	rowsAffected, err := r.base.Updates(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", id)
	}, map[string]any{
		"name":        name,
		"description": description,
		"updated_at":  now,
	})
	if err != nil {
		return nil, fmt.Errorf("update project: %w", err)
	}
	if rowsAffected == 0 {
		return nil, domain.ErrProjectNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *ProjectRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.ProjectRecord{})
	if result.Error != nil {
		return fmt.Errorf("delete project: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrProjectNotFound
	}
	return nil
}

func toDomainProject(record models.ProjectRecord) *domain.Project {
	return &domain.Project{
		ID:          record.ID,
		Name:        record.Name,
		Description: record.Description,
		CreatedBy:   record.CreatedBy,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}
}

var _ domain.ProjectRepository = (*ProjectRepository)(nil)
