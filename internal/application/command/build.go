package command

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"tango/internal/domain"
)

// ── CreateBuildJob ────────────────────────────────────────────────────────────

type CreateBuildJobCommand struct {
	GitURL    string
	GitBranch string
	ImageTag  string
}

type BuildService interface {
	RunAsync(job *domain.BuildJob)
}

type CreateBuildJobHandler struct {
	repo    domain.BuildJobRepository
	builder BuildService
}

func NewCreateBuildJobHandler(repo domain.BuildJobRepository, builder BuildService) *CreateBuildJobHandler {
	return &CreateBuildJobHandler{repo: repo, builder: builder}
}

func (h *CreateBuildJobHandler) Handle(ctx context.Context, cmd CreateBuildJobCommand) (*domain.BuildJob, error) {
	job, err := domain.NewBuildJob(newBuildJobID(), cmd.GitURL, cmd.GitBranch, cmd.ImageTag)
	if err != nil {
		return nil, err
	}
	saved, err := h.repo.Save(ctx, job)
	if err != nil {
		return nil, err
	}
	h.builder.RunAsync(saved)
	return saved, nil
}

func newBuildJobID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "build_" + hex.EncodeToString(b)
}

// ── CancelBuildJob ────────────────────────────────────────────────────────────

type CancelBuildJobCommand struct {
	ID string
}

type CancelBuildJobHandler struct {
	repo domain.BuildJobRepository
}

func NewCancelBuildJobHandler(repo domain.BuildJobRepository) *CancelBuildJobHandler {
	return &CancelBuildJobHandler{repo: repo}
}

func (h *CancelBuildJobHandler) Handle(ctx context.Context, cmd CancelBuildJobCommand) (*domain.BuildJob, error) {
	job, err := h.repo.GetByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}
	if !job.CanCancel() {
		return nil, domain.ErrBuildJobNotCancelable
	}
	job.Status = domain.BuildJobStatusCanceled
	return h.repo.Update(ctx, job)
}
