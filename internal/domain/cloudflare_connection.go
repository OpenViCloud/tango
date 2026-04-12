package domain

import (
	"context"
	"errors"
	"strings"
	"time"
)

type CloudflareConnectionStatus string

const (
	CloudflareConnectionStatusActive  CloudflareConnectionStatus = "active"
	CloudflareConnectionStatusInvalid CloudflareConnectionStatus = "invalid"
)

var (
	ErrCloudflareConnectionNotFound = errors.New("cloudflare connection not found")
)

type CloudflareConnection struct {
	ID                string
	UserID            string
	DisplayName       string
	AccountID         string
	ZoneID            string
	APITokenEncrypted string
	Status            CloudflareConnectionStatus
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func NewCloudflareConnection(
	id string,
	userID string,
	displayName string,
	accountID string,
	zoneID string,
	apiTokenEncrypted string,
) (*CloudflareConnection, error) {
	now := time.Now().UTC()
	item := &CloudflareConnection{
		ID:                strings.TrimSpace(id),
		UserID:            strings.TrimSpace(userID),
		DisplayName:       strings.TrimSpace(displayName),
		AccountID:         strings.TrimSpace(accountID),
		ZoneID:            strings.TrimSpace(zoneID),
		APITokenEncrypted: strings.TrimSpace(apiTokenEncrypted),
		Status:            CloudflareConnectionStatusActive,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := item.Validate(); err != nil {
		return nil, err
	}
	return item, nil
}

func (c *CloudflareConnection) Validate() error {
	if c.ID == "" || c.UserID == "" || c.DisplayName == "" || c.AccountID == "" || c.ZoneID == "" || c.APITokenEncrypted == "" {
		return ErrInvalidInput
	}
	switch c.Status {
	case CloudflareConnectionStatusActive, CloudflareConnectionStatusInvalid:
		return nil
	default:
		return ErrInvalidInput
	}
}

func (c *CloudflareConnection) Update(displayName, accountID, zoneID string) error {
	c.DisplayName = strings.TrimSpace(displayName)
	c.AccountID = strings.TrimSpace(accountID)
	c.ZoneID = strings.TrimSpace(zoneID)
	c.UpdatedAt = time.Now().UTC()
	return c.Validate()
}

func (c *CloudflareConnection) ReplaceEncryptedToken(encrypted string) error {
	c.APITokenEncrypted = strings.TrimSpace(encrypted)
	c.UpdatedAt = time.Now().UTC()
	return c.Validate()
}

type CloudflareConnectionRepository interface {
	Save(ctx context.Context, item *CloudflareConnection) (*CloudflareConnection, error)
	Update(ctx context.Context, item *CloudflareConnection) (*CloudflareConnection, error)
	GetByID(ctx context.Context, id string) (*CloudflareConnection, error)
	ListByUser(ctx context.Context, userID string) ([]*CloudflareConnection, error)
	Delete(ctx context.Context, id string) error
}
