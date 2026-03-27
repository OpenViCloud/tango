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
	ResourceStatusCreating     ResourceStatus = "creating"
	ResourceStatusPulling      ResourceStatus = "pulling"
	ResourceStatusRunning      ResourceStatus = "running"
	ResourceStatusStopped      ResourceStatus = "stopped"
	ResourceStatusError        ResourceStatus = "error"
	ResourceStatusPendingBuild ResourceStatus = "pending_build" // waiting for build job
	ResourceStatusBuilding     ResourceStatus = "building"      // build job running
)

// SourceType describes where the resource image comes from.
const (
	ResourceSourcePreset = "preset" // pre-built Docker image (postgres, redis, etc.)
	ResourceSourceGit    = "git"    // build from a git repository
	ResourceSourceImage  = "image"  // user-supplied image URL
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
	// Source fields
	SourceType string // "preset" | "git" | "image"
	GitURL     string
	GitBranch  string
	BuildMode  string // "auto" | "dockerfile"
	BuildJobID string // populated when SourceType == "git"
	GitToken   string // encrypted access token for private repos
	Ports      []ResourcePort
	EnvVars    []ResourceEnvVar
	CreatedAt  time.Time
	UpdatedAt  time.Time
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
	SourceType    string
	GitURL        string
	GitBranch     string
	BuildMode     string
	BuildJobID    string
	GitToken      string
	Ports         []ResourcePort
	EnvVars       []ResourceEnvVar
}

type UpdateResourceInput struct {
	Name  string
	Ports []ResourcePort
}

type ResourceRepository interface {
	Create(ctx context.Context, input CreateResourceInput) (*Resource, error)
	ListByEnvironment(ctx context.Context, environmentID string) ([]*Resource, error)
	GetByID(ctx context.Context, id string) (*Resource, error)
	Update(ctx context.Context, id string, input UpdateResourceInput) (*Resource, error)
	UpdateStatus(ctx context.Context, id string, status ResourceStatus, containerID string) error
	// UpdateBuildComplete sets the image, tag, build_job_id and transitions status to stopped.
	UpdateBuildComplete(ctx context.Context, id string, image string, buildJobID string) error
	Delete(ctx context.Context, id string) error
	SetEnvVars(ctx context.Context, resourceID string, vars []ResourceEnvVar) error
}
