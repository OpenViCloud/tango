package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"tango/internal/domain"
	"tango/internal/infrastructure/persistence/models"

	"gorm.io/gorm"
)

type ChannelRepository struct {
	db   *gorm.DB
	base *BaseRepository[models.ChannelRecord]
}

func NewChannelRepository(db *gorm.DB) *ChannelRepository {
	return &ChannelRepository{
		db:   db,
		base: NewBaseRepository[models.ChannelRecord](db),
	}
}

func (r *ChannelRepository) Save(ctx context.Context, channel *domain.Channel) (*domain.Channel, error) {
	record := toChannelRecord(channel)
	if err := r.base.Create(ctx, &record); err != nil {
		if isUniqueConstraintError(err) {
			return nil, domain.ErrChannelAlreadyExists
		}
		return nil, fmt.Errorf("save channel: %w", err)
	}
	return r.GetByID(ctx, channel.ID)
}

func (r *ChannelRepository) Update(ctx context.Context, channel *domain.Channel) (*domain.Channel, error) {
	record := toChannelRecord(channel)
	record.UpdatedAt = time.Now().UTC()
	rowsAffected, err := r.base.Updates(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ? AND deleted_at IS NULL", channel.ID)
	}, map[string]any{
		"name":                  record.Name,
		"kind":                  record.Kind,
		"status":                record.Status,
		"encrypted_credentials": record.EncryptedCredentials,
		"settings_json":         record.SettingsJSON,
		"updated_at":            record.UpdatedAt,
	})
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, domain.ErrChannelAlreadyExists
		}
		return nil, fmt.Errorf("update channel: %w", err)
	}
	if rowsAffected == 0 {
		return nil, domain.ErrChannelNotFound
	}
	return r.GetByID(ctx, channel.ID)
}

func (r *ChannelRepository) Delete(ctx context.Context, id string) error {
	now := time.Now().UTC()
	rowsAffected, err := r.base.Updates(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ? AND deleted_at IS NULL", id)
	}, map[string]any{
		"deleted_at": now,
		"updated_at": now,
	})
	if err != nil {
		return fmt.Errorf("delete channel: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrChannelNotFound
	}
	return nil
}

func (r *ChannelRepository) GetByID(ctx context.Context, id string) (*domain.Channel, error) {
	return r.queryOne(ctx, true, func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ? AND deleted_at IS NULL", id)
	})
}

func (r *ChannelRepository) GetByName(ctx context.Context, name string) (*domain.Channel, error) {
	return r.queryOne(ctx, false, func(db *gorm.DB) *gorm.DB {
		return db.Where("LOWER(name) = LOWER(?) AND deleted_at IS NULL", name)
	})
}

func (r *ChannelRepository) GetAll(ctx context.Context, opts domain.ChannelListOptions) (*domain.ChannelListResult, error) {
	records, total, err := r.base.Page(ctx, func(db *gorm.DB) *gorm.DB {
		db = db.Where("deleted_at IS NULL")
		if search := strings.TrimSpace(opts.SearchText); search != "" {
			like := "%" + strings.ToLower(search) + "%"
			db = db.Where(
				"LOWER(name) LIKE ? OR LOWER(kind) LIKE ? OR LOWER(status) LIKE ?",
				like, like, like,
			)
		}
		return db
	}, PageOptions{
		PageIndex: opts.PageIndex,
		PageSize:  opts.PageSize,
		OrderBy:   opts.OrderBy,
		Ascending: opts.Ascending,
		AllowedSort: map[string]string{
			"":           "created_at",
			"createdat":  "created_at",
			"created_at": "created_at",
			"name":       "name",
			"kind":       "kind",
			"status":     "status",
			"updatedat":  "updated_at",
			"updated_at": "updated_at",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("list channels: %w", err)
	}

	items := make([]*domain.Channel, 0, len(records))
	for _, record := range records {
		items = append(items, toDomainChannel(record))
	}
	return &domain.ChannelListResult{Items: items, TotalItems: total}, nil
}

func (r *ChannelRepository) ListByWorkspaceID(ctx context.Context, workspaceID string) ([]*domain.Channel, error) {
	var records []models.ChannelRecord
	if err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND deleted_at IS NULL", workspaceID).
		Order("name ASC").
		Order("created_at ASC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list channels by workspace id: %w", err)
	}

	items := make([]*domain.Channel, 0, len(records))
	for _, record := range records {
		items = append(items, toDomainChannel(record))
	}
	return items, nil
}

func (r *ChannelRepository) ListActiveConfigured(ctx context.Context) ([]*domain.Channel, error) {
	var records []models.ChannelRecord
	if err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL AND status = ? AND encrypted_credentials IS NOT NULL AND encrypted_credentials <> ''", string(domain.ChannelStatusActive)).
		Order("kind ASC").
		Order("updated_at DESC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list active configured channels: %w", err)
	}

	seenKinds := make(map[string]struct{}, len(records))
	items := make([]*domain.Channel, 0, len(records))
	for _, record := range records {
		if _, ok := seenKinds[record.Kind]; ok {
			continue
		}
		seenKinds[record.Kind] = struct{}{}
		items = append(items, toDomainChannel(record))
	}
	return items, nil
}

func (r *ChannelRepository) SetStatusByKindExcept(ctx context.Context, kind domain.ChannelKind, exceptID string, status domain.ChannelStatus) error {
	query := r.db.WithContext(ctx).Model(&models.ChannelRecord{}).
		Where("kind = ? AND deleted_at IS NULL", string(kind))
	if exceptID != "" {
		query = query.Where("id <> ?", exceptID)
	}
	if err := query.Updates(map[string]any{
		"status":     string(status),
		"updated_at": time.Now().UTC(),
	}).Error; err != nil {
		return fmt.Errorf("set status by kind: %w", err)
	}
	return nil
}

func (r *ChannelRepository) queryOne(ctx context.Context, required bool, scope func(*gorm.DB) *gorm.DB) (*domain.Channel, error) {
	var record models.ChannelRecord
	query := scope(r.db.WithContext(ctx).Model(&models.ChannelRecord{}))
	if err := query.First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			if required {
				return nil, domain.ErrChannelNotFound
			}
			return nil, nil
		}
		return nil, err
	}
	return toDomainChannel(record), nil
}

func toChannelRecord(channel *domain.Channel) models.ChannelRecord {
	return models.ChannelRecord{
		ID:                   channel.ID,
		Name:                 channel.Name,
		Kind:                 string(channel.Kind),
		Status:               string(channel.Status),
		EncryptedCredentials: channel.EncryptedCredentials,
		SettingsJSON:         channel.SettingsJSON,
		CreatedAt:            channel.CreatedAt,
		UpdatedAt:            channel.UpdatedAt,
		DeletedAt:            channel.DeletedAt,
	}
}

func toDomainChannel(record models.ChannelRecord) *domain.Channel {
	return &domain.Channel{
		ID:                   record.ID,
		Name:                 record.Name,
		Kind:                 domain.ChannelKind(record.Kind),
		Status:               domain.ChannelStatus(record.Status),
		EncryptedCredentials: record.EncryptedCredentials,
		SettingsJSON:         record.SettingsJSON,
		CreatedAt:            record.CreatedAt,
		UpdatedAt:            record.UpdatedAt,
		DeletedAt:            record.DeletedAt,
	}
}

var _ domain.ChannelRepository = (*ChannelRepository)(nil)

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique constraint") ||
		strings.Contains(message, "duplicate key") ||
		strings.Contains(message, "duplicated key") ||
		strings.Contains(message, "unique failed")
}
