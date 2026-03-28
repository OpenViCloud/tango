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

// ── Fork Environment ──────────────────────────────────────────────────────────

type ForkEnvironmentCommand struct {
	// SourceEnvID is the environment to copy resources from.
	SourceEnvID string
	// NewEnvID is the pre-generated ID for the new environment.
	NewEnvID string
	// Name is the name for the new environment.
	Name string
	// CreatedBy is the user performing the fork.
	CreatedBy string
}

type ForkEnvironmentHandler struct {
	envRepo      domain.EnvironmentRepository
	resourceRepo domain.ResourceRepository
}

func NewForkEnvironmentHandler(envRepo domain.EnvironmentRepository, resourceRepo domain.ResourceRepository) *ForkEnvironmentHandler {
	return &ForkEnvironmentHandler{envRepo: envRepo, resourceRepo: resourceRepo}
}

func (h *ForkEnvironmentHandler) Handle(ctx context.Context, cmd ForkEnvironmentCommand) (*domain.Environment, error) {
	// Load source environment to get the project ID.
	sourceEnv, err := h.envRepo.GetByID(ctx, cmd.SourceEnvID)
	if err != nil {
		return nil, fmt.Errorf("get source environment: %w", err)
	}

	// List source resources (without env vars).
	sourceResources, err := h.resourceRepo.ListByEnvironment(ctx, cmd.SourceEnvID)
	if err != nil {
		return nil, fmt.Errorf("list source resources: %w", err)
	}

	// Create the new environment.
	newEnv, err := h.envRepo.Create(ctx, domain.CreateEnvironmentInput{
		ID:        cmd.NewEnvID,
		Name:      cmd.Name,
		ProjectID: sourceEnv.ProjectID,
	})
	if err != nil {
		return nil, fmt.Errorf("create forked environment: %w", err)
	}

	// Clone each resource with full env vars.
	for _, src := range sourceResources {
		full, err := h.resourceRepo.GetByID(ctx, src.ID)
		if err != nil {
			return nil, fmt.Errorf("get resource %s: %w", src.ID, err)
		}

		ports := make([]domain.ResourcePort, len(full.Ports))
		for i, p := range full.Ports {
			ports[i] = domain.ResourcePort{
				HostPort:     p.HostPort,
				InternalPort: p.InternalPort,
				Proto:        p.Proto,
				Label:        p.Label,
			}
		}

		envVars := make([]domain.ResourceEnvVar, len(full.EnvVars))
		for i, ev := range full.EnvVars {
			envVars[i] = domain.ResourceEnvVar{
				Key:      ev.Key,
				Value:    ev.Value,
				IsSecret: ev.IsSecret,
			}
		}

		if _, err := h.resourceRepo.Create(ctx, domain.CreateResourceInput{
			ID:            newResourceID(),
			Name:          full.Name,
			Type:          full.Type,
			Image:         full.Image,
			Tag:           full.Tag,
			Config:        full.Config,
			EnvironmentID: newEnv.ID,
			CreatedBy:     cmd.CreatedBy,
			SourceType:    full.SourceType,
			GitURL:        full.GitURL,
			GitBranch:     full.GitBranch,
			BuildMode:     full.BuildMode,
			GitToken:      full.GitToken,
			ImageTag:      full.ImageTag,
			ConnectionID:  full.ConnectionID,
			// BuildJobID and ContainerID intentionally omitted (runtime state).
			Ports:   ports,
			EnvVars: envVars,
		}); err != nil {
			return nil, fmt.Errorf("create forked resource %s: %w", full.Name, err)
		}
	}

	return newEnv, nil
}
