package command

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"tango/internal/domain"
)

// CreateAPIKey

type CreateAPIKeyCommand struct {
	ID        string
	Name      string
	UserID    string
	ExpiresAt *time.Time // nil = never expires
}

type CreateAPIKeyResult struct {
	APIKey   *domain.APIKey
	PlainKey string // shown once, never stored
}

type CreateAPIKeyHandler struct {
	repo domain.APIKeyRepository
}

func NewCreateAPIKeyHandler(repo domain.APIKeyRepository) *CreateAPIKeyHandler {
	return &CreateAPIKeyHandler{repo: repo}
}

func (h *CreateAPIKeyHandler) Handle(ctx context.Context, cmd CreateAPIKeyCommand) (*CreateAPIKeyResult, error) {
	plainKey, err := generateSecureKey()
	if err != nil {
		return nil, fmt.Errorf("generate api key: %w", err)
	}
	keyHash := hashKey(plainKey)

	key, err := domain.NewAPIKey(cmd.ID, cmd.Name, keyHash, cmd.UserID, cmd.ExpiresAt)
	if err != nil {
		return nil, err
	}

	saved, err := h.repo.Save(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("save api key: %w", err)
	}

	return &CreateAPIKeyResult{
		APIKey:   saved,
		PlainKey: plainKey,
	}, nil
}

// RevokeAPIKey

type RevokeAPIKeyCommand struct {
	ID     string
	UserID string // must own the key
}

type RevokeAPIKeyHandler struct {
	repo domain.APIKeyRepository
}

func NewRevokeAPIKeyHandler(repo domain.APIKeyRepository) *RevokeAPIKeyHandler {
	return &RevokeAPIKeyHandler{repo: repo}
}

func (h *RevokeAPIKeyHandler) Handle(ctx context.Context, cmd RevokeAPIKeyCommand) error {
	key, err := h.repo.GetByID(ctx, cmd.ID)
	if err != nil {
		return err
	}
	if key.UserID != cmd.UserID {
		return domain.ErrAPIKeyNotFound
	}
	return h.repo.Delete(ctx, cmd.ID)
}

// helpers

func generateSecureKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "tango_" + hex.EncodeToString(b), nil
}

func HashAPIKey(plainKey string) string {
	return hashKey(plainKey)
}

func hashKey(plainKey string) string {
	sum := sha256.Sum256([]byte(plainKey))
	return hex.EncodeToString(sum[:])
}
