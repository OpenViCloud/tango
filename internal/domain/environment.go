package domain

import (
	"context"
	"time"
)

type Environment struct {
	ID        string
	Name      string
	ProjectID string
	CreatedAt time.Time
	Resources []Resource
}

type CreateEnvironmentInput struct {
	ID        string
	Name      string
	ProjectID string
}

type EnvironmentRepository interface {
	Create(ctx context.Context, input CreateEnvironmentInput) (*Environment, error)
	ListByProject(ctx context.Context, projectID string) ([]*Environment, error)
	GetByID(ctx context.Context, id string) (*Environment, error)
	Delete(ctx context.Context, id string) error
}
