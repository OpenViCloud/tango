package command

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"tango/internal/domain"
)

type ResourceStartRunner interface {
	RunStartAsync(run *domain.ResourceRun)
}

type CreateStartResourceRunCommand struct {
	ResourceID string
}

type CreateStartResourceRunHandler struct {
	resourceRepo domain.ResourceRepository
	runRepo      domain.ResourceRunRepository
	runner       ResourceStartRunner
}

func NewCreateStartResourceRunHandler(
	resourceRepo domain.ResourceRepository,
	runRepo domain.ResourceRunRepository,
	runner ResourceStartRunner,
) *CreateStartResourceRunHandler {
	return &CreateStartResourceRunHandler{
		resourceRepo: resourceRepo,
		runRepo:      runRepo,
		runner:       runner,
	}
}

func (h *CreateStartResourceRunHandler) Handle(ctx context.Context, cmd CreateStartResourceRunCommand) (*domain.ResourceRun, error) {
	if _, err := h.resourceRepo.GetByID(ctx, cmd.ResourceID); err != nil {
		return nil, err
	}

	run, err := domain.NewResourceRun(newResourceRunID(), cmd.ResourceID)
	if err != nil {
		return nil, err
	}
	saved, err := h.runRepo.Save(ctx, run)
	if err != nil {
		return nil, err
	}
	h.runner.RunStartAsync(saved)
	return saved, nil
}

func newResourceRunID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "resrun_" + hex.EncodeToString(b)
}
