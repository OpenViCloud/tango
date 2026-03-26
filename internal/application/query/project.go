package query

import (
	"context"

	"tango/internal/domain"
)

// ── List Projects ─────────────────────────────────────────────────────────────

type ListProjectsQuery struct{}

type ListProjectsHandler struct {
	projectRepo domain.ProjectRepository
	envRepo     domain.EnvironmentRepository
}

func NewListProjectsHandler(projectRepo domain.ProjectRepository, envRepo domain.EnvironmentRepository) *ListProjectsHandler {
	return &ListProjectsHandler{projectRepo: projectRepo, envRepo: envRepo}
}

func (h *ListProjectsHandler) Handle(ctx context.Context, q ListProjectsQuery) ([]*domain.Project, error) {
	projects, err := h.projectRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, p := range projects {
		envs, err := h.envRepo.ListByProject(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		p.Environments = make([]domain.Environment, 0, len(envs))
		for _, e := range envs {
			p.Environments = append(p.Environments, *e)
		}
	}
	return projects, nil
}

// ── Get Project ───────────────────────────────────────────────────────────────

type GetProjectQuery struct {
	ID string
}

type GetProjectHandler struct {
	projectRepo  domain.ProjectRepository
	envRepo      domain.EnvironmentRepository
	resourceRepo domain.ResourceRepository
}

func NewGetProjectHandler(projectRepo domain.ProjectRepository, envRepo domain.EnvironmentRepository, resourceRepo domain.ResourceRepository) *GetProjectHandler {
	return &GetProjectHandler{projectRepo: projectRepo, envRepo: envRepo, resourceRepo: resourceRepo}
}

func (h *GetProjectHandler) Handle(ctx context.Context, q GetProjectQuery) (*domain.Project, error) {
	project, err := h.projectRepo.GetByID(ctx, q.ID)
	if err != nil {
		return nil, err
	}
	envs, err := h.envRepo.ListByProject(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	project.Environments = make([]domain.Environment, 0, len(envs))
	for _, e := range envs {
		resources, err := h.resourceRepo.ListByEnvironment(ctx, e.ID)
		if err != nil {
			return nil, err
		}
		e.Resources = make([]domain.Resource, 0, len(resources))
		for _, r := range resources {
			e.Resources = append(e.Resources, *r)
		}
		project.Environments = append(project.Environments, *e)
	}
	return project, nil
}

// ── List Environment Resources ────────────────────────────────────────────────

type ListEnvironmentResourcesQuery struct {
	EnvironmentID string
}

type ListEnvironmentResourcesHandler struct {
	resourceRepo domain.ResourceRepository
}

func NewListEnvironmentResourcesHandler(resourceRepo domain.ResourceRepository) *ListEnvironmentResourcesHandler {
	return &ListEnvironmentResourcesHandler{resourceRepo: resourceRepo}
}

func (h *ListEnvironmentResourcesHandler) Handle(ctx context.Context, q ListEnvironmentResourcesQuery) ([]*domain.Resource, error) {
	return h.resourceRepo.ListByEnvironment(ctx, q.EnvironmentID)
}

// ── Get Resource ──────────────────────────────────────────────────────────────

type GetResourceQuery struct {
	ID string
}

type GetResourceHandler struct {
	resourceRepo domain.ResourceRepository
}

func NewGetResourceHandler(resourceRepo domain.ResourceRepository) *GetResourceHandler {
	return &GetResourceHandler{resourceRepo: resourceRepo}
}

func (h *GetResourceHandler) Handle(ctx context.Context, q GetResourceQuery) (*domain.Resource, error) {
	return h.resourceRepo.GetByID(ctx, q.ID)
}
