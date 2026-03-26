package repositories

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type PageOptions struct {
	PageIndex   int
	PageSize    int
	OrderBy     string
	Ascending   bool
	AllowedSort map[string]string
}

type BaseRepository[T any] struct {
	db *gorm.DB
}

func NewBaseRepository[T any](db *gorm.DB) *BaseRepository[T] {
	return &BaseRepository[T]{db: db}
}

func (r *BaseRepository[T]) Create(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

func (r *BaseRepository[T]) First(ctx context.Context, dest *T, scope func(*gorm.DB) *gorm.DB) error {
	query := r.db.WithContext(ctx).Model(new(T))
	if scope != nil {
		query = scope(query)
	}
	return query.First(dest).Error
}

func (r *BaseRepository[T]) Updates(ctx context.Context, scope func(*gorm.DB) *gorm.DB, values any) (int64, error) {
	query := r.db.WithContext(ctx).Model(new(T))
	if scope != nil {
		query = scope(query)
	}
	result := query.Updates(values)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (r *BaseRepository[T]) Page(ctx context.Context, scope func(*gorm.DB) *gorm.DB, opts PageOptions) ([]T, int64, error) {
	query := r.db.WithContext(ctx).Model(new(T))
	if scope != nil {
		query = scope(query)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count records: %w", err)
	}

	pageIndex := opts.PageIndex
	if pageIndex < 0 {
		pageIndex = 0
	}
	pageSize := opts.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	orderColumn := "created_at"
	if column, ok := opts.AllowedSort[strings.ToLower(strings.TrimSpace(opts.OrderBy))]; ok {
		orderColumn = column
	}
	direction := "DESC"
	if opts.Ascending {
		direction = "ASC"
	}

	var items []T
	if err := query.
		Order(orderColumn + " " + direction).
		Limit(pageSize).
		Offset(pageIndex * pageSize).
		Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("query page: %w", err)
	}
	return items, total, nil
}
