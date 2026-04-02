package services

import (
	"context"
	"log/slog"
	"strings"

	appservices "tango/internal/application/services"
	"tango/internal/domain"
)

type ResourceRuntimeReconciler struct {
	resourceRepo domain.ResourceRepository
	dockerRepo   domain.DockerRepository
	logger       *slog.Logger
}

func NewResourceRuntimeReconciler(
	resourceRepo domain.ResourceRepository,
	dockerRepo domain.DockerRepository,
	logger *slog.Logger,
) *ResourceRuntimeReconciler {
	return &ResourceRuntimeReconciler{
		resourceRepo: resourceRepo,
		dockerRepo:   dockerRepo,
		logger:       logger,
	}
}

func (r *ResourceRuntimeReconciler) ReconcileAll(ctx context.Context) (*appservices.ResourceRuntimeReconcileSummary, error) {
	resources, err := r.resourceRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	return r.reconcile(ctx, resources)
}

func (r *ResourceRuntimeReconciler) ReconcileResource(ctx context.Context, resourceID string) (*appservices.ResourceRuntimeReconcileSummary, error) {
	resource, err := r.resourceRepo.GetByID(ctx, resourceID)
	if err != nil {
		return nil, err
	}
	return r.reconcile(ctx, []*domain.Resource{resource})
}

func (r *ResourceRuntimeReconciler) reconcile(ctx context.Context, resources []*domain.Resource) (*appservices.ResourceRuntimeReconcileSummary, error) {
	summary := &appservices.ResourceRuntimeReconcileSummary{}
	if r.dockerRepo == nil {
		return summary, nil
	}

	containers, err := r.dockerRepo.ListContainers(ctx, true)
	if err != nil {
		return nil, err
	}

	containerByID := make(map[string]domain.Container, len(containers))
	for _, container := range containers {
		containerByID[container.ID] = container
	}

	for _, resource := range resources {
		if shouldSkipRuntimeReconcile(resource) {
			continue
		}

		summary.Checked++

		nextStatus := resource.Status
		nextContainerID := resource.ContainerID

		if strings.TrimSpace(resource.ContainerID) == "" {
			if resource.Status == domain.ResourceStatusRunning {
				nextStatus = domain.ResourceStatusStopped
			}
		} else if container, ok := containerByID[resource.ContainerID]; !ok {
			nextStatus = domain.ResourceStatusStopped
			nextContainerID = ""
			summary.MissingContainers++
		} else {
			nextStatus = mapContainerStateToResourceStatus(container.State)
		}

		switch nextStatus {
		case domain.ResourceStatusRunning:
			summary.Running++
		case domain.ResourceStatusError:
			summary.Errored++
		default:
			summary.Stopped++
		}

		if nextStatus == resource.Status && nextContainerID == resource.ContainerID {
			continue
		}

		if err := r.resourceRepo.UpdateStatus(ctx, resource.ID, nextStatus, nextContainerID); err != nil {
			return nil, err
		}
		summary.Updated++
		r.logger.Info("resource runtime reconciled",
			"resource_id", resource.ID,
			"from_status", resource.Status,
			"to_status", nextStatus,
			"from_container_id", resource.ContainerID,
			"to_container_id", nextContainerID,
		)
	}

	return summary, nil
}

func shouldSkipRuntimeReconcile(resource *domain.Resource) bool {
	switch resource.Status {
	case domain.ResourceStatusBuilding, domain.ResourceStatusPendingBuild, domain.ResourceStatusCreated:
		return strings.TrimSpace(resource.ContainerID) == ""
	default:
		return false
	}
}

func mapContainerStateToResourceStatus(state string) domain.ResourceStatus {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "running", "restarting":
		return domain.ResourceStatusRunning
	case "dead":
		return domain.ResourceStatusError
	default:
		return domain.ResourceStatusStopped
	}
}
