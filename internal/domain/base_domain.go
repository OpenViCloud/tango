package domain

import (
	"context"
	"errors"
	"time"
)

var ErrBaseDomainNotFound = errors.New("base domain not found")
var ErrBaseDomainConflict = errors.New("base domain already exists")

type BaseDomain struct {
	ID              string
	Domain          string
	WildcardEnabled bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type BaseDomainRepository interface {
	Create(ctx context.Context, bd BaseDomain) (*BaseDomain, error)
	List(ctx context.Context) ([]*BaseDomain, error)
	GetByID(ctx context.Context, id string) (*BaseDomain, error)
	GetByDomain(ctx context.Context, domain string) (*BaseDomain, error)
	Delete(ctx context.Context, id string) error
}
