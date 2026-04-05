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
	ResourceStatusCreated      ResourceStatus = "created"       // saved but never built
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
	TLSEnabled    bool
	// Source fields
	SourceType   string // "preset" | "git" | "image"
	GitURL       string
	GitBranch    string
	BuildMode    string // "auto" | "dockerfile"
	BuildJobID   string // populated when SourceType == "git"
	GitToken     string // encrypted access token for private repos
	ImageTag     string // target registry image tag for builds
	ConnectionID string // source connection ID for private repos
	// Cluster fields
	NodeID      *string // swarm node ID for placement constraint; nil = any node
	Replicas    int     // number of swarm service replicas; 0/1 = single instance
	MemoryLimit int64   // memory hard limit in bytes; 0 = unlimited
	CPULimit    int64   // CPU limit in nanoCPUs (1e9 = 1 core); 0 = unlimited
	Ports    []ResourcePort
	EnvVars []ResourceEnvVar
	CreatedAt time.Time
	UpdatedAt time.Time
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
	TLSEnabled    bool
	SourceType    string
	GitURL        string
	GitBranch     string
	BuildMode     string
	BuildJobID    string
	GitToken      string
	ImageTag      string
	ConnectionID  string
	NodeID        *string // swarm placement constraint
	Replicas      int     // number of swarm replicas; 0/1 = single instance
	MemoryLimit   int64   // bytes; 0 = unlimited
	CPULimit      int64   // nanoCPUs; 0 = unlimited
	Ports         []ResourcePort
	EnvVars       []ResourceEnvVar
}

type UpdateResourceInput struct {
	Name        string
	Ports       []ResourcePort
	TLSEnabled  bool
	Config      map[string]any
	Replicas    int   // swarm replica count; 0/1 = single instance
	MemoryLimit int64 // bytes; 0 = unlimited
	CPULimit    int64 // nanoCPUs; 0 = unlimited
}

type ResourceRepository interface {
	Create(ctx context.Context, input CreateResourceInput) (*Resource, error)
	ListAll(ctx context.Context) ([]*Resource, error)
	ListByEnvironment(ctx context.Context, environmentID string) ([]*Resource, error)
	GetByID(ctx context.Context, id string) (*Resource, error)
	Update(ctx context.Context, id string, input UpdateResourceInput) (*Resource, error)
	UpdateStatus(ctx context.Context, id string, status ResourceStatus, containerID string) error
	// UpdateBuildComplete sets the image, tag, build_job_id and transitions status to stopped.
	UpdateBuildComplete(ctx context.Context, id string, image string, buildJobID string) error
	Delete(ctx context.Context, id string) error
	SetEnvVars(ctx context.Context, resourceID string, vars []ResourceEnvVar) error
	// FindRunningByHostPort returns the running resource that already owns the given
	// host port, or nil when the port is free.
	FindRunningByHostPort(ctx context.Context, hostPort int) (*Resource, error)
}
