package query

import (
	"context"

	"tango/internal/domain"
)

// ── List Containers ───────────────────────────────────────────────────────────

type ListContainersQuery struct {
	All bool
}

type ListContainersHandler struct {
	docker domain.DockerRepository
}

func NewListContainersHandler(docker domain.DockerRepository) *ListContainersHandler {
	return &ListContainersHandler{docker: docker}
}

func (h *ListContainersHandler) Handle(ctx context.Context, q ListContainersQuery) ([]domain.Container, error) {
	return h.docker.ListContainers(ctx, q.All)
}

// ── List Images ───────────────────────────────────────────────────────────────

type ListImagesHandler struct {
	docker domain.DockerRepository
}

func NewListImagesHandler(docker domain.DockerRepository) *ListImagesHandler {
	return &ListImagesHandler{docker: docker}
}

func (h *ListImagesHandler) Handle(ctx context.Context) ([]domain.Image, error) {
	return h.docker.ListImages(ctx)
}

// ── Container Details ─────────────────────────────────────────────────────────

type GetContainerDetailsHandler struct {
	docker domain.DockerRepository
}

func NewGetContainerDetailsHandler(docker domain.DockerRepository) *GetContainerDetailsHandler {
	return &GetContainerDetailsHandler{docker: docker}
}

func (h *GetContainerDetailsHandler) Handle(ctx context.Context, containerID string) (domain.ContainerDetails, error) {
	return h.docker.GetContainerDetails(ctx, containerID)
}

// ── Container Stats ───────────────────────────────────────────────────────────

type GetContainerStatsHandler struct {
	docker domain.DockerRepository
}

func NewGetContainerStatsHandler(docker domain.DockerRepository) *GetContainerStatsHandler {
	return &GetContainerStatsHandler{docker: docker}
}

func (h *GetContainerStatsHandler) Handle(ctx context.Context, containerID string) (domain.ContainerStats, error) {
	return h.docker.GetContainerStats(ctx, containerID)
}
