package repositories

import (
	"context"
	"fmt"
	"time"

	"tango/internal/domain"
	"tango/internal/infrastructure/persistence/models"

	"gorm.io/gorm"
)

type ServerRepository struct {
	db   *gorm.DB
	base *BaseRepository[models.ServerRecord]
}

func NewServerRepository(db *gorm.DB) *ServerRepository {
	return &ServerRepository{
		db:   db,
		base: NewBaseRepository[models.ServerRecord](db),
	}
}

func (r *ServerRepository) Save(ctx context.Context, s *domain.Server) (*domain.Server, error) {
	record := toServerRecord(s)
	if err := r.base.Create(ctx, &record); err != nil {
		return nil, fmt.Errorf("save server: %w", err)
	}
	return r.GetByID(ctx, s.ID)
}

func (r *ServerRepository) Update(ctx context.Context, s *domain.Server) (*domain.Server, error) {
	record := toServerRecord(s)
	record.UpdatedAt = time.Now().UTC()
	rowsAffected, err := r.base.Updates(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", s.ID)
	}, record)
	if err != nil {
		return nil, fmt.Errorf("update server: %w", err)
	}
	if rowsAffected == 0 {
		return nil, domain.ErrServerNotFound
	}
	return r.GetByID(ctx, s.ID)
}

func (r *ServerRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.ServerRecord{})
	if result.Error != nil {
		return fmt.Errorf("delete server: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrServerNotFound
	}
	return nil
}

func (r *ServerRepository) GetByID(ctx context.Context, id string) (*domain.Server, error) {
	var record models.ServerRecord
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrServerNotFound
		}
		return nil, err
	}
	return toDomainServer(record), nil
}

func (r *ServerRepository) ListAll(ctx context.Context) ([]*domain.Server, error) {
	var records []models.ServerRecord
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list servers: %w", err)
	}
	items := make([]*domain.Server, 0, len(records))
	for _, rec := range records {
		items = append(items, toDomainServer(rec))
	}
	return items, nil
}

func toServerRecord(s *domain.Server) models.ServerRecord {
	return models.ServerRecord{
		ID:         s.ID,
		Name:       s.Name,
		PublicIP:   s.PublicIP,
		PrivateIP:  s.PrivateIP,
		SSHUser:    s.SSHUser,
		SSHPort:    s.SSHPort,
		Status:     string(s.Status),
		ErrorMsg:   s.ErrorMsg,
		LastPingAt: s.LastPingAt,
		CreatedAt:  s.CreatedAt,
		UpdatedAt:  s.UpdatedAt,
	}
}

func toDomainServer(r models.ServerRecord) *domain.Server {
	return &domain.Server{
		ID:         r.ID,
		Name:       r.Name,
		PublicIP:   r.PublicIP,
		PrivateIP:  r.PrivateIP,
		SSHUser:    r.SSHUser,
		SSHPort:    r.SSHPort,
		Status:     domain.ServerStatus(r.Status),
		ErrorMsg:   r.ErrorMsg,
		LastPingAt: r.LastPingAt,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
}

var _ domain.ServerRepository = (*ServerRepository)(nil)
