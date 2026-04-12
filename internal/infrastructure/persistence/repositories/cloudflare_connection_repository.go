package repositories

import (
	"context"
	"fmt"

	"tango/internal/domain"
	"tango/internal/infrastructure/persistence/models"

	"gorm.io/gorm"
)

type CloudflareConnectionRepository struct {
	db *gorm.DB
}

func NewCloudflareConnectionRepository(db *gorm.DB) *CloudflareConnectionRepository {
	return &CloudflareConnectionRepository{db: db}
}

func (r *CloudflareConnectionRepository) Save(ctx context.Context, item *domain.CloudflareConnection) (*domain.CloudflareConnection, error) {
	record := toCloudflareConnectionRecord(item)
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, fmt.Errorf("save cloudflare connection: %w", err)
	}
	return r.GetByID(ctx, item.ID)
}

func (r *CloudflareConnectionRepository) Update(ctx context.Context, item *domain.CloudflareConnection) (*domain.CloudflareConnection, error) {
	record := toCloudflareConnectionRecord(item)
	result := r.db.WithContext(ctx).
		Model(&models.CloudflareConnectionRecord{}).
		Where("id = ?", item.ID).
		Updates(map[string]any{
			"display_name":        record.DisplayName,
			"account_id":          record.AccountID,
			"zone_id":             record.ZoneID,
			"api_token_encrypted": record.APITokenEncrypted,
			"status":              record.Status,
			"updated_at":          record.UpdatedAt,
		})
	if result.Error != nil {
		return nil, fmt.Errorf("update cloudflare connection: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, domain.ErrCloudflareConnectionNotFound
	}
	return r.GetByID(ctx, item.ID)
}

func (r *CloudflareConnectionRepository) GetByID(ctx context.Context, id string) (*domain.CloudflareConnection, error) {
	var record models.CloudflareConnectionRecord
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrCloudflareConnectionNotFound
		}
		return nil, fmt.Errorf("get cloudflare connection by id: %w", err)
	}
	return toDomainCloudflareConnection(record), nil
}

func (r *CloudflareConnectionRepository) ListByUser(ctx context.Context, userID string) ([]*domain.CloudflareConnection, error) {
	var records []models.CloudflareConnectionRecord
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list cloudflare connections: %w", err)
	}
	items := make([]*domain.CloudflareConnection, 0, len(records))
	for _, record := range records {
		items = append(items, toDomainCloudflareConnection(record))
	}
	return items, nil
}

func (r *CloudflareConnectionRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.CloudflareConnectionRecord{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete cloudflare connection: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrCloudflareConnectionNotFound
	}
	return nil
}

func toCloudflareConnectionRecord(item *domain.CloudflareConnection) models.CloudflareConnectionRecord {
	return models.CloudflareConnectionRecord{
		ID:                item.ID,
		UserID:            item.UserID,
		DisplayName:       item.DisplayName,
		AccountID:         item.AccountID,
		ZoneID:            item.ZoneID,
		APITokenEncrypted: item.APITokenEncrypted,
		Status:            string(item.Status),
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	}
}

func toDomainCloudflareConnection(record models.CloudflareConnectionRecord) *domain.CloudflareConnection {
	return &domain.CloudflareConnection{
		ID:                record.ID,
		UserID:            record.UserID,
		DisplayName:       record.DisplayName,
		AccountID:         record.AccountID,
		ZoneID:            record.ZoneID,
		APITokenEncrypted: record.APITokenEncrypted,
		Status:            domain.CloudflareConnectionStatus(record.Status),
		CreatedAt:         record.CreatedAt,
		UpdatedAt:         record.UpdatedAt,
	}
}

var _ domain.CloudflareConnectionRepository = (*CloudflareConnectionRepository)(nil)
