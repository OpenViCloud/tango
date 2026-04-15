package query

import (
	"context"

	"tango/internal/domain"
)

// ListAPIKeys

type ListAPIKeysHandler struct {
	repo domain.APIKeyRepository
}

func NewListAPIKeysHandler(repo domain.APIKeyRepository) *ListAPIKeysHandler {
	return &ListAPIKeysHandler{repo: repo}
}

func (h *ListAPIKeysHandler) Handle(ctx context.Context, userID string) ([]*domain.APIKey, error) {
	return h.repo.ListByUserID(ctx, userID)
}

// FindAPIKeyByHash — used by auth middleware

type FindAPIKeyByHashHandler struct {
	repo domain.APIKeyRepository
}

func NewFindAPIKeyByHashHandler(repo domain.APIKeyRepository) *FindAPIKeyByHashHandler {
	return &FindAPIKeyByHashHandler{repo: repo}
}

func (h *FindAPIKeyByHashHandler) Handle(ctx context.Context, keyHash string) (*domain.APIKey, error) {
	return h.repo.FindByHash(ctx, keyHash)
}
