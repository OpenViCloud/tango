package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrProjectNotFound     = errors.New("project not found")
	ErrEnvironmentNotFound = errors.New("environment not found")
	ErrResourceNotFound    = errors.New("resource not found")
	ErrResourceNotStarted  = errors.New("resource container has not been created yet")
)

type Project struct {
	ID           string
	Name         string
	Description  string
	CreatedBy    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Environments []Environment
}

type CreateProjectInput struct {
	ID          string
	Name        string
	Description string
	CreatedBy   string
}

type ProjectRepository interface {
	Create(ctx context.Context, input CreateProjectInput) (*Project, error)
	List(ctx context.Context) ([]*Project, error)
	GetByID(ctx context.Context, id string) (*Project, error)
	Update(ctx context.Context, id, name, description string) (*Project, error)
	Delete(ctx context.Context, id string) error
}
