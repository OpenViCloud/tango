package domain

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type SourceProviderStatus string

const (
	SourceProviderStatusActive  SourceProviderStatus = "active"
	SourceProviderStatusInvalid SourceProviderStatus = "invalid"
)

var (
	ErrSourceProviderNotFound         = errors.New("source provider not found")
	ErrSourceProviderStatusInvalid    = errors.New("source provider status is invalid")
	ErrSourceProviderEncryptionFailed = errors.New("source provider encryption failed")
)

type SourceProvider struct {
	ID                   string
	UserID               string
	Provider             SourceConnectionProvider
	DisplayName          string
	EncryptedCredentials string
	MetadataJSON         string
	Status               SourceProviderStatus
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func NewSourceProvider(
	id string,
	userID string,
	provider string,
	displayName string,
	encryptedCredentials string,
	metadataJSON string,
	status string,
) (*SourceProvider, error) {
	now := time.Now().UTC()
	item := &SourceProvider{
		ID:                   strings.TrimSpace(id),
		UserID:               strings.TrimSpace(userID),
		Provider:             SourceConnectionProvider(strings.TrimSpace(strings.ToLower(provider))),
		DisplayName:          strings.TrimSpace(displayName),
		EncryptedCredentials: strings.TrimSpace(encryptedCredentials),
		MetadataJSON:         normalizeJSON(metadataJSON),
		Status:               SourceProviderStatus(strings.TrimSpace(strings.ToLower(status))),
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if err := item.Validate(); err != nil {
		return nil, err
	}
	return item, nil
}

func (p *SourceProvider) Validate() error {
	if strings.TrimSpace(p.ID) == "" ||
		strings.TrimSpace(p.UserID) == "" ||
		strings.TrimSpace(p.DisplayName) == "" ||
		strings.TrimSpace(p.EncryptedCredentials) == "" {
		return ErrInvalidInput
	}

	switch p.Provider {
	case SourceConnectionProviderGitHub:
	default:
		return ErrSourceConnectionProviderInvalid
	}

	switch p.Status {
	case SourceProviderStatusActive, SourceProviderStatusInvalid:
	default:
		return ErrSourceProviderStatusInvalid
	}

	if !json.Valid([]byte(normalizeJSON(p.MetadataJSON))) {
		return ErrInvalidInput
	}

	return nil
}

type SourceProviderRepository interface {
	Save(ctx context.Context, provider *SourceProvider) (*SourceProvider, error)
	GetByID(ctx context.Context, id string) (*SourceProvider, error)
	GetByUserAndProvider(ctx context.Context, userID string, provider SourceConnectionProvider) (*SourceProvider, error)
}
