package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"tango/internal/domain"
	"tango/internal/infrastructure/persistence/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ResourceRepository struct {
	db   *gorm.DB
	base *BaseRepository[models.ResourceRecord]
}

func NewResourceRepository(db *gorm.DB) *ResourceRepository {
	return &ResourceRepository{
		db:   db,
		base: NewBaseRepository[models.ResourceRecord](db),
	}
}

func (r *ResourceRepository) Create(ctx context.Context, input domain.CreateResourceInput) (*domain.Resource, error) {
	configJSON, err := marshalConfig(input.Config)
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}

	sourceType := input.SourceType
	if sourceType == "" {
		sourceType = domain.ResourceSourcePreset
	}
	now := time.Now().UTC()
	record := models.ResourceRecord{
		ID:            input.ID,
		Name:          input.Name,
		Type:          input.Type,
		Status:        domain.ResourceStatusCreating,
		ContainerID:   "",
		Image:         input.Image,
		Tag:           input.Tag,
		Config:        configJSON,
		EnvironmentID: input.EnvironmentID,
		CreatedBy:     input.CreatedBy,
		SourceType:    sourceType,
		GitURL:        input.GitURL,
		GitBranch:     input.GitBranch,
		BuildMode:     input.BuildMode,
		BuildJobID:    input.BuildJobID,
		GitToken:      input.GitToken,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&record).Error; err != nil {
			return fmt.Errorf("create resource record: %w", err)
		}

		for _, p := range input.Ports {
			portRecord := models.ResourcePortRecord{
				ID:           uuid.New().String(),
				ResourceID:   input.ID,
				HostPort:     p.HostPort,
				InternalPort: p.InternalPort,
				Proto:        p.Proto,
				Label:        p.Label,
			}
			if portRecord.Proto == "" {
				portRecord.Proto = "tcp"
			}
			if err := tx.Create(&portRecord).Error; err != nil {
				return fmt.Errorf("create resource port record: %w", err)
			}
		}

		for _, ev := range input.EnvVars {
			evRecord := models.ResourceEnvVarRecord{
				ID:         uuid.New().String(),
				ResourceID: input.ID,
				Key:        ev.Key,
				Value:      ev.Value,
				IsSecret:   ev.IsSecret,
				CreatedAt:  now,
			}
			if err := tx.Create(&evRecord).Error; err != nil {
				return fmt.Errorf("create resource env var record: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return r.GetByID(ctx, input.ID)
}

func (r *ResourceRepository) ListByEnvironment(ctx context.Context, environmentID string) ([]*domain.Resource, error) {
	var records []models.ResourceRecord
	if err := r.db.WithContext(ctx).Where("environment_id = ?", environmentID).Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list resources by environment: %w", err)
	}

	items := make([]*domain.Resource, 0, len(records))
	for _, rec := range records {
		var portRecords []models.ResourcePortRecord
		if err := r.db.WithContext(ctx).Where("resource_id = ?", rec.ID).Find(&portRecords).Error; err != nil {
			return nil, fmt.Errorf("list resource ports: %w", err)
		}

		res, err := toDomainResource(rec, portRecords, nil)
		if err != nil {
			return nil, err
		}
		items = append(items, res)
	}
	return items, nil
}

func (r *ResourceRepository) GetByID(ctx context.Context, id string) (*domain.Resource, error) {
	var record models.ResourceRecord
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrResourceNotFound
		}
		return nil, fmt.Errorf("get resource by id: %w", err)
	}

	var portRecords []models.ResourcePortRecord
	if err := r.db.WithContext(ctx).Where("resource_id = ?", id).Find(&portRecords).Error; err != nil {
		return nil, fmt.Errorf("list resource ports: %w", err)
	}

	var evRecords []models.ResourceEnvVarRecord
	if err := r.db.WithContext(ctx).Where("resource_id = ?", id).Find(&evRecords).Error; err != nil {
		return nil, fmt.Errorf("list resource env vars: %w", err)
	}

	return toDomainResource(record, portRecords, evRecords)
}

func (r *ResourceRepository) Update(ctx context.Context, id string, input domain.UpdateResourceInput) (*domain.Resource, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.ResourceRecord{}).Where("id = ?", id).Updates(map[string]any{
			"name":         input.Name,
			"container_id": "",
			"updated_at":   time.Now().UTC(),
		})
		if result.Error != nil {
			return fmt.Errorf("update resource: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return domain.ErrResourceNotFound
		}

		if err := tx.Where("resource_id = ?", id).Delete(&models.ResourcePortRecord{}).Error; err != nil {
			return fmt.Errorf("delete resource ports: %w", err)
		}
		for _, p := range input.Ports {
			proto := p.Proto
			if proto == "" {
				proto = "tcp"
			}
			portRecord := models.ResourcePortRecord{
				ID:           uuid.New().String(),
				ResourceID:   id,
				HostPort:     p.HostPort,
				InternalPort: p.InternalPort,
				Proto:        proto,
				Label:        p.Label,
			}
			if err := tx.Create(&portRecord).Error; err != nil {
				return fmt.Errorf("create resource port: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

func (r *ResourceRepository) UpdateBuildComplete(ctx context.Context, id string, image string, buildJobID string) error {
	result := r.db.WithContext(ctx).Model(&models.ResourceRecord{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"image":        image,
			"tag":          "",
			"build_job_id": buildJobID,
			"status":       domain.ResourceStatusStopped,
			"updated_at":   time.Now().UTC(),
		})
	if result.Error != nil {
		return fmt.Errorf("update resource build complete: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrResourceNotFound
	}
	return nil
}

func (r *ResourceRepository) UpdateStatus(ctx context.Context, id string, status domain.ResourceStatus, containerID string) error {
	result := r.db.WithContext(ctx).Model(&models.ResourceRecord{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":       status,
			"container_id": containerID,
			"updated_at":   time.Now().UTC(),
		})
	if result.Error != nil {
		return fmt.Errorf("update resource status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrResourceNotFound
	}
	return nil
}

func (r *ResourceRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.ResourceRecord{})
	if result.Error != nil {
		return fmt.Errorf("delete resource: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrResourceNotFound
	}
	return nil
}

func (r *ResourceRepository) SetEnvVars(ctx context.Context, resourceID string, vars []domain.ResourceEnvVar) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("resource_id = ?", resourceID).Delete(&models.ResourceEnvVarRecord{}).Error; err != nil {
			return fmt.Errorf("delete env vars: %w", err)
		}
		for _, v := range vars {
			record := models.ResourceEnvVarRecord{
				ID:         uuid.New().String(),
				ResourceID: resourceID,
				Key:        v.Key,
				Value:      v.Value,
				IsSecret:   v.IsSecret,
				CreatedAt:  now,
			}
			if err := tx.Create(&record).Error; err != nil {
				return fmt.Errorf("create env var: %w", err)
			}
		}
		return nil
	})
}

func toDomainResource(record models.ResourceRecord, portRecords []models.ResourcePortRecord, evRecords []models.ResourceEnvVarRecord) (*domain.Resource, error) {
	config, err := unmarshalConfig(record.Config)
	if err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	ports := make([]domain.ResourcePort, 0, len(portRecords))
	for _, p := range portRecords {
		ports = append(ports, domain.ResourcePort{
			ID:           p.ID,
			ResourceID:   p.ResourceID,
			HostPort:     p.HostPort,
			InternalPort: p.InternalPort,
			Proto:        p.Proto,
			Label:        p.Label,
		})
	}

	envVars := make([]domain.ResourceEnvVar, 0, len(evRecords))
	for _, ev := range evRecords {
		envVars = append(envVars, domain.ResourceEnvVar{
			ID:         ev.ID,
			ResourceID: ev.ResourceID,
			Key:        ev.Key,
			Value:      ev.Value,
			IsSecret:   ev.IsSecret,
		})
	}

	return &domain.Resource{
		ID:            record.ID,
		Name:          record.Name,
		Type:          record.Type,
		Status:        record.Status,
		ContainerID:   record.ContainerID,
		Image:         record.Image,
		Tag:           record.Tag,
		Config:        config,
		EnvironmentID: record.EnvironmentID,
		CreatedBy:     record.CreatedBy,
		SourceType:    record.SourceType,
		GitURL:        record.GitURL,
		GitBranch:     record.GitBranch,
		BuildMode:     record.BuildMode,
		BuildJobID:    record.BuildJobID,
		GitToken:      record.GitToken,
		Ports:         ports,
		EnvVars:       envVars,
		CreatedAt:     record.CreatedAt,
		UpdatedAt:     record.UpdatedAt,
	}, nil
}

func marshalConfig(config map[string]any) (string, error) {
	if len(config) == 0 {
		return "{}", nil
	}
	b, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func unmarshalConfig(s string) (map[string]any, error) {
	if s == "" || s == "{}" {
		return map[string]any{}, nil
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return nil, err
	}
	return result, nil
}

var _ domain.ResourceRepository = (*ResourceRepository)(nil)
