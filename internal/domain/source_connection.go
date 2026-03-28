package domain

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type SourceConnectionProvider string
type SourceConnectionStatus string

const (
	SourceConnectionProviderGitHub SourceConnectionProvider = "github"

	SourceConnectionStatusActive  SourceConnectionStatus = "active"
	SourceConnectionStatusInvalid SourceConnectionStatus = "invalid"
)

var (
	ErrSourceConnectionNotFound          = errors.New("source connection not found")
	ErrSourceConnectionProviderInvalid   = errors.New("source connection provider is invalid")
	ErrSourceConnectionStatusInvalid     = errors.New("source connection status is invalid")
	ErrSourceConnectionEncryptionFailed  = errors.New("source connection encryption failed")
	ErrSourceConnectionCredentialsAbsent = errors.New("source connection credentials are missing")
	ErrSourceConnectionOAuthStateInvalid = errors.New("source connection oauth state is invalid")
)

type SourceConnection struct {
	ID                string
	UserID            string
	SourceProviderID  string
	Provider          SourceConnectionProvider
	DisplayName       string
	AccountIdentifier string
	ExternalID        string
	MetadataJSON      string
	Status            SourceConnectionStatus
	ExpiresAt         *time.Time
	LastUsedAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func NewSourceConnection(
	id string,
	userID string,
	provider string,
	sourceProviderID string,
	displayName string,
	accountIdentifier string,
	externalID string,
	metadataJSON string,
	status string,
	expiresAt *time.Time,
) (*SourceConnection, error) {
	now := time.Now().UTC()
	connection := &SourceConnection{
		ID:                strings.TrimSpace(id),
		UserID:            strings.TrimSpace(userID),
		SourceProviderID:  strings.TrimSpace(sourceProviderID),
		Provider:          SourceConnectionProvider(strings.TrimSpace(strings.ToLower(provider))),
		DisplayName:       strings.TrimSpace(displayName),
		AccountIdentifier: strings.TrimSpace(accountIdentifier),
		ExternalID:        strings.TrimSpace(externalID),
		MetadataJSON:      normalizeJSON(metadataJSON),
		Status:            SourceConnectionStatus(strings.TrimSpace(strings.ToLower(status))),
		ExpiresAt:         expiresAt,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := connection.Validate(); err != nil {
		return nil, err
	}
	return connection, nil
}

func (c *SourceConnection) Validate() error {
	if strings.TrimSpace(c.ID) == "" ||
		strings.TrimSpace(c.UserID) == "" ||
		strings.TrimSpace(c.SourceProviderID) == "" ||
		strings.TrimSpace(c.DisplayName) == "" ||
		strings.TrimSpace(c.AccountIdentifier) == "" ||
		strings.TrimSpace(c.ExternalID) == "" {
		return ErrInvalidInput
	}

	switch c.Provider {
	case SourceConnectionProviderGitHub:
	default:
		return ErrSourceConnectionProviderInvalid
	}

	switch c.Status {
	case SourceConnectionStatusActive, SourceConnectionStatusInvalid:
	default:
		return ErrSourceConnectionStatusInvalid
	}

	if !json.Valid([]byte(normalizeJSON(c.MetadataJSON))) {
		return ErrInvalidInput
	}

	return nil
}

func normalizeJSON(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "{}"
	}
	return trimmed
}

type SourceConnectionRepository interface {
	Save(ctx context.Context, connection *SourceConnection) (*SourceConnection, error)
	GetByID(ctx context.Context, id string) (*SourceConnection, error)
	GetByProviderAndAccount(ctx context.Context, userID string, provider SourceConnectionProvider, accountIdentifier string) (*SourceConnection, error)
	ListByUser(ctx context.Context, userID string) ([]*SourceConnection, error)
	TouchUsedAt(ctx context.Context, id string, usedAt time.Time) error
}
