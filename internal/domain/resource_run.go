package domain

import (
	"context"
	"errors"
	"strings"
	"time"
)

type ResourceRunStatus string

const (
	ResourceRunStatusQueued        ResourceRunStatus = "queued"
	ResourceRunStatusCheckingImage ResourceRunStatus = "checking_image"
	ResourceRunStatusPullingImage  ResourceRunStatus = "pulling_image"
	ResourceRunStatusCreating      ResourceRunStatus = "creating_container"
	ResourceRunStatusStarting      ResourceRunStatus = "starting_container"
	ResourceRunStatusDone          ResourceRunStatus = "done"
	ResourceRunStatusFailed        ResourceRunStatus = "failed"
)

var (
	ErrResourceRunNotFound = errors.New("resource run not found")
)

type ResourceRun struct {
	ID         string
	ResourceID string
	Status     ResourceRunStatus
	Logs       string
	ErrorMsg   string
	StartedAt  *time.Time
	FinishedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func NewResourceRun(id, resourceID string) (*ResourceRun, error) {
	if strings.TrimSpace(id) == "" {
		return nil, errors.New("resource run id is required")
	}
	if strings.TrimSpace(resourceID) == "" {
		return nil, errors.New("resource run resource_id is required")
	}
	return &ResourceRun{
		ID:         strings.TrimSpace(id),
		ResourceID: strings.TrimSpace(resourceID),
		Status:     ResourceRunStatusQueued,
	}, nil
}

type ResourceRunRepository interface {
	Save(ctx context.Context, run *ResourceRun) (*ResourceRun, error)
	Update(ctx context.Context, run *ResourceRun) (*ResourceRun, error)
	GetByID(ctx context.Context, id string) (*ResourceRun, error)
}
