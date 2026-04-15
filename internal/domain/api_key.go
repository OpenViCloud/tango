package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrAPIKeyNotFound    = errors.New("api key not found")
	ErrAPIKeyExpired     = errors.New("api key expired")
	ErrAPIKeyNameEmpty   = errors.New("api key name is required")
)

type APIKey struct {
	ID         string
	Name       string
	KeyHash    string
	UserID     string
	ExpiresAt  *time.Time // nil = never expires
	LastUsedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func NewAPIKey(id, name, keyHash, userID string, expiresAt *time.Time) (*APIKey, error) {
	if name == "" {
		return nil, ErrAPIKeyNameEmpty
	}
	now := time.Now().UTC()
	return &APIKey{
		ID:        id,
		Name:      name,
		KeyHash:   keyHash,
		UserID:    userID,
		ExpiresAt: expiresAt,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().UTC().After(*k.ExpiresAt)
}

type APIKeyRepository interface {
	Save(ctx context.Context, key *APIKey) (*APIKey, error)
	FindByHash(ctx context.Context, keyHash string) (*APIKey, error)
	GetByID(ctx context.Context, id string) (*APIKey, error)
	ListByUserID(ctx context.Context, userID string) ([]*APIKey, error)
	UpdateLastUsed(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
}
