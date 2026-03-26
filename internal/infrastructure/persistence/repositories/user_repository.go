package repositories

import (
	"context"
	"fmt"
	"strings"
	"time"

	"tango/internal/domain"
	"tango/internal/infrastructure/persistence/models"

	"gorm.io/gorm"
)

type UserRepository struct {
	db   *gorm.DB
	base *BaseRepository[models.UserRecord]
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		db:   db,
		base: NewBaseRepository[models.UserRecord](db),
	}
}

func (r *UserRepository) Save(ctx context.Context, user *domain.User) (*domain.User, error) {
	record := toUserRecord(user)
	if err := r.base.Create(ctx, &record); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}
	return r.GetByID(ctx, user.ID)
}

func (r *UserRepository) Update(ctx context.Context, user *domain.User) (*domain.User, error) {
	record := toUserRecord(user)
	record.UpdatedAt = time.Now().UTC()
	rowsAffected, err := r.base.Updates(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", user.ID)
	}, record)
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	if rowsAffected == 0 {
		return nil, domain.ErrUserNotFound
	}
	return r.GetByID(ctx, user.ID)
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	now := time.Now().UTC()
	rowsAffected, err := r.base.Updates(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ? AND deleted_at IS NULL", id)
	}, map[string]any{
		"deleted_at": now,
		"updated_at": now,
	})
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	return r.queryOne(ctx, false, func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ? AND deleted_at IS NULL", id)
	})
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.queryOne(ctx, false, func(db *gorm.DB) *gorm.DB {
		return db.Where("LOWER(email) = LOWER(?) AND deleted_at IS NULL", email)
	})
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return r.queryOne(ctx, true, func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ? AND deleted_at IS NULL", id)
	})
}

func (r *UserRepository) GetAll(ctx context.Context, opts domain.UserListOptions) (*domain.UserListResult, error) {
	records, total, err := r.base.Page(ctx, func(db *gorm.DB) *gorm.DB {
		db = db.Where("deleted_at IS NULL")
		if search := strings.TrimSpace(opts.SearchText); search != "" {
			like := "%" + strings.ToLower(search) + "%"
			db = db.Where(
				"LOWER(email) LIKE ? OR LOWER(nickname) LIKE ? OR LOWER(first_name) LIKE ? OR LOWER(last_name) LIKE ?",
				like, like, like, like,
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
			"email":      "email",
			"nickname":   "nickname",
			"firstname":  "first_name",
			"first_name": "first_name",
			"lastname":   "last_name",
			"last_name":  "last_name",
			"status":     "status",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	items := make([]*domain.User, 0, len(records))
	for _, record := range records {
		items = append(items, toDomainUser(record))
	}
	return &domain.UserListResult{
		Items:      items,
		TotalItems: total,
	}, nil
}

func (r *UserRepository) queryOne(ctx context.Context, required bool, scope func(*gorm.DB) *gorm.DB) (*domain.User, error) {
	var record models.UserRecord
	query := scope(r.db.WithContext(ctx).Model(&models.UserRecord{}))
	if err := query.First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			if required {
				return nil, domain.ErrUserNotFound
			}
			return nil, nil
		}
		return nil, err
	}
	return toDomainUser(record), nil
}

func toUserRecord(user *domain.User) models.UserRecord {
	return models.UserRecord{
		ID:           user.ID,
		Email:        user.Email,
		Nickname:     user.Nickname,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		Phone:        user.Phone,
		Address:      user.Address,
		PasswordHash: user.PasswordHash,
		Status:       string(user.Status),
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		DeletedAt:    user.DeletedAt,
	}
}

func toDomainUser(record models.UserRecord) *domain.User {
	return &domain.User{
		ID:           record.ID,
		Email:        record.Email,
		Nickname:     record.Nickname,
		FirstName:    record.FirstName,
		LastName:     record.LastName,
		Phone:        record.Phone,
		Address:      record.Address,
		PasswordHash: record.PasswordHash,
		Status:       domain.UserStatus(record.Status),
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
		DeletedAt:    record.DeletedAt,
	}
}

var _ domain.UserRepository = (*UserRepository)(nil)
