package repositories

import (
	"context"
	"fmt"
	"strings"
	"time"

	"tango/internal/domain"
	"tango/internal/infrastructure/persistence/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RoleRepository struct {
	db   *gorm.DB
	base *BaseRepository[models.RoleRecord]
}

func NewRoleRepository(db *gorm.DB) *RoleRepository {
	return &RoleRepository{
		db:   db,
		base: NewBaseRepository[models.RoleRecord](db),
	}
}

func (r *RoleRepository) EnsureRole(ctx context.Context, role *domain.Role) error {
	if err := role.Validate(); err != nil {
		return err
	}
	record := models.RoleRecord{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		IsSystem:    role.IsSystem,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	}
	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "name"}},
			DoUpdates: clause.AssignmentColumns([]string{"description", "is_system", "updated_at"}),
		}).
		Create(&record).Error; err != nil {
		return fmt.Errorf("ensure role: %w", err)
	}
	return nil
}

func (r *RoleRepository) Save(ctx context.Context, role *domain.Role) (*domain.Role, error) {
	if err := role.Validate(); err != nil {
		return nil, err
	}
	record := toRoleRecord(role)
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, fmt.Errorf("save role: %w", err)
	}
	return r.GetByID(ctx, role.ID)
}

func (r *RoleRepository) Update(ctx context.Context, role *domain.Role) (*domain.Role, error) {
	if err := role.Validate(); err != nil {
		return nil, err
	}
	record := toRoleRecord(role)
	record.UpdatedAt = time.Now().UTC()
	rowsAffected, err := r.base.Updates(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", role.ID)
	}, map[string]any{
		"name":        record.Name,
		"description": record.Description,
		"is_system":   record.IsSystem,
		"updated_at":  record.UpdatedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("update role: %w", err)
	}
	if rowsAffected == 0 {
		return nil, domain.ErrRoleNotFound
	}
	return r.GetByID(ctx, role.ID)
}

func (r *RoleRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.RoleRecord{})
	if result.Error != nil {
		return fmt.Errorf("delete role: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrRoleNotFound
	}
	return nil
}

func (r *RoleRepository) GetByID(ctx context.Context, id string) (*domain.Role, error) {
	return r.queryOne(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", id)
	})
}

func (r *RoleRepository) GetByName(ctx context.Context, name string) (*domain.Role, error) {
	return r.queryOne(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("LOWER(name) = LOWER(?)", strings.TrimSpace(name))
	})
}

func (r *RoleRepository) GetAll(ctx context.Context, opts domain.RoleListOptions) (*domain.RoleListResult, error) {
	records, total, err := r.base.Page(ctx, func(db *gorm.DB) *gorm.DB {
		if search := strings.TrimSpace(opts.SearchText); search != "" {
			like := "%" + strings.ToLower(search) + "%"
			db = db.Where(
				"LOWER(name) LIKE ? OR LOWER(description) LIKE ?",
				like, like,
			)
		}
		return db
	}, PageOptions{
		PageIndex: opts.PageIndex,
		PageSize:  opts.PageSize,
		OrderBy:   opts.OrderBy,
		Ascending: opts.Ascending,
		AllowedSort: map[string]string{
			"":            "created_at",
			"createdat":   "created_at",
			"created_at":  "created_at",
			"name":        "name",
			"description": "description",
			"system":      "is_system",
			"is_system":   "is_system",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}

	items := make([]*domain.Role, 0, len(records))
	for _, record := range records {
		items = append(items, toDomainRole(record))
	}
	return &domain.RoleListResult{Items: items, TotalItems: total}, nil
}

func (r *RoleRepository) ListByUserID(ctx context.Context, userID string) ([]*domain.Role, error) {
	var records []models.RoleRecord
	if err := r.db.WithContext(ctx).
		Model(&models.RoleRecord{}).
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Order("roles.name ASC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list roles by user: %w", err)
	}

	items := make([]*domain.Role, 0, len(records))
	for _, record := range records {
		items = append(items, toDomainRole(record))
	}
	return items, nil
}

func (r *RoleRepository) AssignRoleToUser(ctx context.Context, userID, roleID string) error {
	record := models.UserRoleRecord{
		UserID:    userID,
		RoleID:    roleID,
		CreatedAt: time.Now().UTC(),
	}
	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&record).Error; err != nil {
		return fmt.Errorf("assign role to user: %w", err)
	}
	return nil
}

func (r *RoleRepository) RemoveRoleFromUser(ctx context.Context, userID, roleID string) error {
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Delete(&models.UserRoleRecord{}).Error; err != nil {
		return fmt.Errorf("remove role from user: %w", err)
	}
	return nil
}

func (r *RoleRepository) queryOne(ctx context.Context, scope func(*gorm.DB) *gorm.DB) (*domain.Role, error) {
	var record models.RoleRecord
	query := scope(r.db.WithContext(ctx).Model(&models.RoleRecord{}))
	if err := query.First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrRoleNotFound
		}
		return nil, err
	}
	return toDomainRole(record), nil
}

func toRoleRecord(role *domain.Role) models.RoleRecord {
	return models.RoleRecord{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		IsSystem:    role.IsSystem,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	}
}

func toDomainRole(record models.RoleRecord) *domain.Role {
	return &domain.Role{
		ID:          record.ID,
		Name:        record.Name,
		Description: record.Description,
		IsSystem:    record.IsSystem,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}
}

var _ domain.RoleRepository = (*RoleRepository)(nil)
