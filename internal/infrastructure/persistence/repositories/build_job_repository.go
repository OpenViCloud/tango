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

type BuildJobRepository struct {
	db *gorm.DB
}

func NewBuildJobRepository(db *gorm.DB) *BuildJobRepository {
	return &BuildJobRepository{db: db}
}

func (r *BuildJobRepository) Save(ctx context.Context, job *domain.BuildJob) (*domain.BuildJob, error) {
	now := time.Now().UTC()
	job.CreatedAt = now
	job.UpdatedAt = now

	record := toBuildJobRecord(job)
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, fmt.Errorf("save build job: %w", err)
	}
	return r.GetByID(ctx, job.ID)
}

func (r *BuildJobRepository) Update(ctx context.Context, job *domain.BuildJob) (*domain.BuildJob, error) {
	job.UpdatedAt = time.Now().UTC()
	result := r.db.WithContext(ctx).
		Model(&models.BuildJobRecord{}).
		Where("id = ?", job.ID).
		Updates(map[string]any{
			"status":       string(job.Status),
			"source_type":  job.SourceType,
			"build_mode":   job.BuildMode,
			"archive_path": job.ArchivePath,
			"archive_name": job.ArchiveName,
			"logs":         job.Logs,
			"error_msg":    job.ErrorMsg,
			"image_tag":    job.ImageTag,
			"started_at":   job.StartedAt,
			"finished_at":  job.FinishedAt,
			"updated_at":   job.UpdatedAt,
		})
	if result.Error != nil {
		return nil, fmt.Errorf("update build job: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, domain.ErrBuildJobNotFound
	}
	return r.GetByID(ctx, job.ID)
}

func (r *BuildJobRepository) GetByID(ctx context.Context, id string) (*domain.BuildJob, error) {
	var record models.BuildJobRecord
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrBuildJobNotFound
		}
		return nil, fmt.Errorf("get build job: %w", err)
	}
	return toDomainBuildJob(&record), nil
}

func (r *BuildJobRepository) List(ctx context.Context, opts domain.BuildJobListOptions) (*domain.BuildJobListResult, error) {
	pageSize := opts.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	pageIndex := opts.PageIndex
	if pageIndex < 0 {
		pageIndex = 0
	}

	q := r.db.WithContext(ctx).Model(&models.BuildJobRecord{})
	if opts.Status != "" {
		q = q.Where("status = ?", opts.Status)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count build jobs: %w", err)
	}

	var records []models.BuildJobRecord
	if err := q.Order("created_at DESC").
		Offset(pageIndex * pageSize).
		Limit(pageSize).
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list build jobs: %w", err)
	}

	items := make([]*domain.BuildJob, 0, len(records))
	for i := range records {
		items = append(items, toDomainBuildJob(&records[i]))
	}
	return &domain.BuildJobListResult{
		Items:      items,
		TotalItems: total,
		PageIndex:  pageIndex,
		PageSize:   pageSize,
	}, nil
}

// ── mapping ──────────────────────────────────────────────────────────────────

func toBuildJobRecord(j *domain.BuildJob) models.BuildJobRecord {
	return models.BuildJobRecord{
		ID:          j.ID,
		Status:      string(j.Status),
		SourceType:  j.SourceType,
		BuildMode:   j.BuildMode,
		GitURL:      j.GitURL,
		GitBranch:   j.GitBranch,
		ArchivePath: j.ArchivePath,
		ArchiveName: j.ArchiveName,
		ImageTag:    j.ImageTag,
		ResourceID:  j.ResourceID,
		Logs:        j.Logs,
		ErrorMsg:    j.ErrorMsg,
		StartedAt:   j.StartedAt,
		FinishedAt:  j.FinishedAt,
		CreatedAt:   j.CreatedAt,
		UpdatedAt:   j.UpdatedAt,
	}
}

func toDomainBuildJob(r *models.BuildJobRecord) *domain.BuildJob {
	return &domain.BuildJob{
		ID:          r.ID,
		Status:      domain.BuildJobStatus(r.Status),
		SourceType:  r.SourceType,
		BuildMode:   r.BuildMode,
		GitURL:      r.GitURL,
		GitBranch:   r.GitBranch,
		ArchivePath: r.ArchivePath,
		ArchiveName: r.ArchiveName,
		ImageTag:    r.ImageTag,
		ResourceID:  r.ResourceID,
		Logs:        r.Logs,
		ErrorMsg:    r.ErrorMsg,
		StartedAt:   r.StartedAt,
		FinishedAt:  r.FinishedAt,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}
