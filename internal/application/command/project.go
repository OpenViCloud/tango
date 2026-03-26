package command

import (
	"context"
	"fmt"

	"tango/internal/domain"
)

// ── Create Project ────────────────────────────────────────────────────────────

type CreateProjectCommand struct {
	ID          string
	Name        string
	Description string
	CreatedBy   string
}

type CreateProjectHandler struct {
	repo domain.ProjectRepository
}

func NewCreateProjectHandler(repo domain.ProjectRepository) *CreateProjectHandler {
	return &CreateProjectHandler{repo: repo}
}

func (h *CreateProjectHandler) Handle(ctx context.Context, cmd CreateProjectCommand) (*domain.Project, error) {
	project, err := h.repo.Create(ctx, domain.CreateProjectInput{
		ID:          cmd.ID,
		Name:        cmd.Name,
		Description: cmd.Description,
		CreatedBy:   cmd.CreatedBy,
	})
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}
	return project, nil
}

// ── Update Project ────────────────────────────────────────────────────────────

type UpdateProjectCommand struct {
	ID          string
	Name        string
	Description string
}

type UpdateProjectHandler struct {
	repo domain.ProjectRepository
}

func NewUpdateProjectHandler(repo domain.ProjectRepository) *UpdateProjectHandler {
	return &UpdateProjectHandler{repo: repo}
}

func (h *UpdateProjectHandler) Handle(ctx context.Context, cmd UpdateProjectCommand) (*domain.Project, error) {
	project, err := h.repo.Update(ctx, cmd.ID, cmd.Name, cmd.Description)
	if err != nil {
		return nil, fmt.Errorf("update project: %w", err)
	}
	return project, nil
}

// ── Delete Project ────────────────────────────────────────────────────────────

type DeleteProjectCommand struct {
	ID string
}

type DeleteProjectHandler struct {
	repo domain.ProjectRepository
}

func NewDeleteProjectHandler(repo domain.ProjectRepository) *DeleteProjectHandler {
	return &DeleteProjectHandler{repo: repo}
}

func (h *DeleteProjectHandler) Handle(ctx context.Context, cmd DeleteProjectCommand) error {
	return h.repo.Delete(ctx, cmd.ID)
}

// ── Create Environment ────────────────────────────────────────────────────────

type CreateEnvironmentCommand struct {
	ID        string
	Name      string
	ProjectID string
}

type CreateEnvironmentHandler struct {
	repo domain.EnvironmentRepository
}

func NewCreateEnvironmentHandler(repo domain.EnvironmentRepository) *CreateEnvironmentHandler {
	return &CreateEnvironmentHandler{repo: repo}
}

func (h *CreateEnvironmentHandler) Handle(ctx context.Context, cmd CreateEnvironmentCommand) (*domain.Environment, error) {
	env, err := h.repo.Create(ctx, domain.CreateEnvironmentInput{
		ID:        cmd.ID,
		Name:      cmd.Name,
		ProjectID: cmd.ProjectID,
	})
	if err != nil {
		return nil, fmt.Errorf("create environment: %w", err)
	}
	return env, nil
}

// ── Delete Environment ────────────────────────────────────────────────────────

type DeleteEnvironmentCommand struct {
	ID string
}

type DeleteEnvironmentHandler struct {
	repo domain.EnvironmentRepository
}

func NewDeleteEnvironmentHandler(repo domain.EnvironmentRepository) *DeleteEnvironmentHandler {
	return &DeleteEnvironmentHandler{repo: repo}
}

func (h *DeleteEnvironmentHandler) Handle(ctx context.Context, cmd DeleteEnvironmentCommand) error {
	return h.repo.Delete(ctx, cmd.ID)
}
