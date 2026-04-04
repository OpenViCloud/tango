package domain

import (
	"context"
	"errors"
	"io"
)

var (
	ErrContainerNotFound = errors.New("container not found")
	ErrImageNotFound     = errors.New("image not found")
)

// Image represents a Docker image.
type Image struct {
	ID      string
	Tags    []string
	Size    int64
	Created int64
	Digest  string
	InUse   int64
}

// ContainerPort represents a network port exposed by a container.
type ContainerPort struct {
	IP          string
	PrivatePort uint16
	PublicPort  uint16
	Type        string
}

// Container represents a Docker container.
type Container struct {
	ID      string
	Name    string
	Image   string
	ImageID string
	State   string
	Status  string
	Command string
	Ports   []ContainerPort
	Labels  map[string]string
}

type ContainerMount struct {
	Type        string
	Name        string
	Source      string
	Destination string
	Driver      string
	Mode        string
	RW          bool
}

type ContainerDetails struct {
	ID           string
	Name         string
	Image        string
	ImageID      string
	Command      []string
	CreatedAt    string
	State        string
	Status       string
	StartedAt    string
	FinishedAt   string
	ExitCode     int
	Error        string
	RestartCount int
	Ports        []ContainerPort
	Labels       map[string]string
	Networks     map[string]string
	Mounts       []ContainerMount
}

type ContainerStats struct {
	ReadAt           string
	CPUPercent       float64
	MemoryUsageBytes uint64
	MemoryLimitBytes uint64
	MemoryPercent    float64
	NetworkRxBytes   uint64
	NetworkTxBytes   uint64
	BlockReadBytes   uint64
	BlockWriteBytes  uint64
	PidsCurrent      uint64
}

// PullImageInput is the input for pulling an image from a registry.
type PullImageInput struct {
	Reference string
}

// CreateContainerInput is the input for creating a new container.
type CreateContainerInput struct {
	Name         string
	Image        string
	Cmd          []string
	TTY          bool
	OpenStdin    bool
	Env          map[string]string
	ExposedPorts []string
	PortBindings map[string]string // containerPort -> hostPort
	Volumes      []string          // host:container bind mounts, e.g. "/data:/data"
	Labels       map[string]string // container labels, e.g. Traefik routing rules
	Networks     []string          // Docker networks to join, e.g. ["traefik-net"]
	AutoRemove   bool
}

type GetContainerLogsInput struct {
	Tail string
}

type ContainerExecInput struct {
	Shell []string
	Cols  uint
	Rows  uint
}

type ContainerExecSession interface {
	io.ReadWriteCloser
	Resize(ctx context.Context, cols, rows uint) error
}

// ContainerInfo holds runtime details about a container returned by InspectContainer.
type ContainerInfo struct {
	ID       string
	Name     string            // container name without leading "/"
	Networks map[string]string // network name → IP address
}

// SwarmNode represents a node in a Docker Swarm cluster.
type SwarmNode struct {
	ID           string
	Hostname     string
	Role         string // "manager" | "worker"
	State        string // "ready" | "down" | ...
	Availability string // "active" | "pause" | "drain"
	ManagerAddr  string // non-empty for manager nodes
}

// CreateServiceInput is the input for creating a Docker Swarm service.
type CreateServiceInput struct {
	Name     string
	Image    string
	Cmd      []string
	Env      map[string]string
	Volumes  []string // host:container bind mounts
	Networks []string // overlay network names
	NodeID   string   // placement constraint — empty means any node
}

// SwarmService represents a running Docker Swarm service.
type SwarmService struct {
	ID   string
	Name string
}

// SwarmRepository manages Docker Swarm services and cluster state.
type SwarmRepository interface {
	// IsManager reports whether the local Docker daemon is an active swarm manager.
	IsManager(ctx context.Context) bool
	// CreateService creates and starts a swarm service.
	CreateService(ctx context.Context, input CreateServiceInput) (SwarmService, error)
	// RemoveService removes a swarm service (stops all its tasks).
	RemoveService(ctx context.Context, serviceID string) error
	// EnsureOverlayNetwork creates an overlay network if it does not already exist.
	EnsureOverlayNetwork(ctx context.Context, name string) error
	// ListNodes returns all nodes in the swarm cluster.
	ListNodes(ctx context.Context) ([]SwarmNode, error)
}

// DockerRepository abstracts all Docker Engine operations.
type DockerRepository interface {
	ListImages(ctx context.Context) ([]Image, error)
	PullImage(ctx context.Context, input PullImageInput) error
	RemoveImage(ctx context.Context, imageID string, force bool) error
	ListContainers(ctx context.Context, all bool) ([]Container, error)
	GetContainerDetails(ctx context.Context, containerID string) (ContainerDetails, error)
	GetContainerStats(ctx context.Context, containerID string) (ContainerStats, error)
	EnsureNetwork(ctx context.Context, name string) error
	CreateContainer(ctx context.Context, input CreateContainerInput) (Container, error)
	InspectContainer(ctx context.Context, containerID string) (ContainerInfo, error)
	GetContainerLogs(ctx context.Context, containerID string, input GetContainerLogsInput) ([]string, error)
	ExecContainer(ctx context.Context, containerID string, input ContainerExecInput) (ContainerExecSession, error)
	StartContainer(ctx context.Context, containerID string) error
	StopContainer(ctx context.Context, containerID string) error
	RemoveContainer(ctx context.Context, containerID string, force bool) error
	Close() error
}
