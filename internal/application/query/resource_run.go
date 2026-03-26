package query

import (
	"context"

	"tango/internal/domain"
)

type GetResourceRunHandler struct {
	repo domain.ResourceRunRepository
}

func NewGetResourceRunHandler(repo domain.ResourceRunRepository) *GetResourceRunHandler {
	return &GetResourceRunHandler{repo: repo}
}

func (h *GetResourceRunHandler) Handle(ctx context.Context, id string) (*domain.ResourceRun, error) {
	return h.repo.GetByID(ctx, id)
}
