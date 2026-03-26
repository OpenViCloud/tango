package domain

import (
	"context"
	"time"
)

type ResourceType = string
type ResourceStatus = string

const (
	ResourceTypeDB      ResourceType = "db"
	ResourceTypeApp     ResourceType = "app"
	ResourceTypeService ResourceType = "service"
)

const (
	ResourceStatusCreating ResourceStatus = "creating"
	ResourceStatusPulling  ResourceStatus = "pulling"
	ResourceStatusRunning  ResourceStatus = "running"
	ResourceStatusStopped  ResourceStatus = "stopped"
	ResourceStatusError    ResourceStatus = "error"
)

type ResourcePort struct {
	ID           string
	ResourceID   string
	HostPort     int
	InternalPort int
	Proto        string
	Label        string
}

type ResourceEnvVar struct {
	ID         string
	ResourceID string
	Key        string
	Value      string
	IsSecret   bool
}

type Resource struct {
	ID            string
	Name          string
	Type          ResourceType
	Status        ResourceStatus
	ContainerID   string
	Image         string
	Tag           string
	Config        map[string]any
	EnvironmentID string
	CreatedBy     string
	Ports         []ResourcePort
	EnvVars       []ResourceEnvVar
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type CreateResourceInput struct {
	ID            string
	Name          string
	Type          ResourceType
	Image         string
	Tag           string
	Config        map[string]any
	EnvironmentID string
	CreatedBy     string
	Ports         []ResourcePort
	EnvVars       []ResourceEnvVar
}

type ResourceRepository interface {
	Create(ctx context.Context, input CreateResourceInput) (*Resource, error)
	ListByEnvironment(ctx context.Context, environmentID string) ([]*Resource, error)
	GetByID(ctx context.Context, id string) (*Resource, error)
	UpdateStatus(ctx context.Context, id string, status ResourceStatus, containerID string) error
	Delete(ctx context.Context, id string) error
	SetEnvVars(ctx context.Context, resourceID string, vars []ResourceEnvVar) error
}
