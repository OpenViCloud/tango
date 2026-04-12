package query

import (
	"context"
	"fmt"
	"strings"

	"tango/internal/domain"
)

type GetCloudflareConnectionQuery struct {
	UserID string
	ID     string
}

type GetCloudflareConnectionHandler struct {
	repo domain.CloudflareConnectionRepository
}

func NewGetCloudflareConnectionHandler(repo domain.CloudflareConnectionRepository) *GetCloudflareConnectionHandler {
	return &GetCloudflareConnectionHandler{repo: repo}
}

func (h *GetCloudflareConnectionHandler) Handle(ctx context.Context, q GetCloudflareConnectionQuery) (*domain.CloudflareConnection, error) {
	item, err := h.repo.GetByID(ctx, strings.TrimSpace(q.ID))
	if err != nil {
		return nil, fmt.Errorf("get cloudflare connection: %w", err)
	}
	if item.UserID != strings.TrimSpace(q.UserID) {
		return nil, domain.ErrCloudflareConnectionNotFound
	}
	return item, nil
}

type ListCloudflareConnectionsQuery struct {
	UserID string
}

type ListCloudflareConnectionsHandler struct {
	repo domain.CloudflareConnectionRepository
}

func NewListCloudflareConnectionsHandler(repo domain.CloudflareConnectionRepository) *ListCloudflareConnectionsHandler {
	return &ListCloudflareConnectionsHandler{repo: repo}
}

func (h *ListCloudflareConnectionsHandler) Handle(ctx context.Context, q ListCloudflareConnectionsQuery) ([]*domain.CloudflareConnection, error) {
	items, err := h.repo.ListByUser(ctx, strings.TrimSpace(q.UserID))
	if err != nil {
		return nil, fmt.Errorf("list cloudflare connections: %w", err)
	}
	return items, nil
}
