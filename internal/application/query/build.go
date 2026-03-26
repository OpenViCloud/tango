package query

import (
	"context"

	"tango/internal/domain"
)

// ── GetBuildJob ───────────────────────────────────────────────────────────────

type GetBuildJobHandler struct {
	repo domain.BuildJobRepository
}

func NewGetBuildJobHandler(repo domain.BuildJobRepository) *GetBuildJobHandler {
	return &GetBuildJobHandler{repo: repo}
}

func (h *GetBuildJobHandler) Handle(ctx context.Context, id string) (*domain.BuildJob, error) {
	return h.repo.GetByID(ctx, id)
}

// ── ListBuildJobs ─────────────────────────────────────────────────────────────

type ListBuildJobsQuery struct {
	PageIndex int
	PageSize  int
	Status    string
}

type ListBuildJobsHandler struct {
	repo domain.BuildJobRepository
}

func NewListBuildJobsHandler(repo domain.BuildJobRepository) *ListBuildJobsHandler {
	return &ListBuildJobsHandler{repo: repo}
}

func (h *ListBuildJobsHandler) Handle(ctx context.Context, q ListBuildJobsQuery) (*domain.BuildJobListResult, error) {
	return h.repo.List(ctx, domain.BuildJobListOptions{
		PageIndex: q.PageIndex,
		PageSize:  q.PageSize,
		Status:    q.Status,
	})
}
