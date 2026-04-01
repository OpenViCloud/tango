package domain

import (
	"context"
	"strings"
	"time"
)

type StorageType string

const (
	StorageTypeLocal StorageType = "local"
	StorageTypeS3    StorageType = "s3"
	StorageTypeMinIO StorageType = "minio"
)

type Storage struct {
	ID                   string
	Name                 string
	Type                 StorageType
	Config               map[string]any
	CredentialsEncrypted string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type CreateStorageInput struct {
	ID                   string
	Name                 string
	Type                 StorageType
	Config               map[string]any
	CredentialsEncrypted string
}

type UpdateStorageInput struct {
	Name                 string
	Type                 StorageType
	Config               map[string]any
	CredentialsEncrypted string
}

func ValidateStorageType(value string) (StorageType, error) {
	switch StorageType(strings.TrimSpace(value)) {
	case StorageTypeLocal, StorageTypeS3, StorageTypeMinIO:
		return StorageType(strings.TrimSpace(value)), nil
	default:
		return "", ErrInvalidInput
	}
}

type StorageRepository interface {
	Create(ctx context.Context, input CreateStorageInput) (*Storage, error)
	GetByID(ctx context.Context, id string) (*Storage, error)
	List(ctx context.Context) ([]*Storage, error)
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, id string, input UpdateStorageInput) (*Storage, error)
}
