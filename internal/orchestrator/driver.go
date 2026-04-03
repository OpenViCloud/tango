package orchestrator

import (
	"context"
	"io"
	"time"
)

// ServiceStatus represents the current state of a managed service.
type ServiceStatus struct {
	Name        string        `json:"name"`
	State       string        `json:"state"`  // "running", "exited", "restarting", "dead", "paused", "created"
	Health      string        `json:"health"` // "healthy", "unhealthy", "starting", "none"
	ContainerID string        `json:"container_id"`
	Image       string        `json:"image"`
	Uptime      time.Duration `json:"uptime"`
	ExitCode    int           `json:"exit_code"`
	Ports       []string      `json:"ports"`
}

// DriverInfo describes the orchestrator backend.
type DriverInfo struct {
	Name    string `json:"name"` // "compose", "k3s", "nomad", "swarm"
	Version string `json:"version"`
	Ready   bool   `json:"ready"`
	Error   string `json:"error,omitempty"`
}

// Driver is the pluggable interface for orchestrator backends.
// Implementations: compose (Phase 1), k3s, swarm, nomad (later).
type Driver interface {
	// Info returns driver metadata and readiness.
	Info(ctx context.Context) (DriverInfo, error)

	// Ping checks if the orchestrator backend is reachable.
	Ping(ctx context.Context) error

	// Up ensures the managed stack is started.
	Up(ctx context.Context) error

	// ListServices returns the status of all managed services.
	ListServices(ctx context.Context) ([]ServiceStatus, error)

	// WaitReady blocks until the target services are running and healthy.
	WaitReady(ctx context.Context, services []string) error

	// ServiceStatus returns the status of a single service by name.
	ServiceStatus(ctx context.Context, name string) (ServiceStatus, error)

	// RestartService restarts a service.
	RestartService(ctx context.Context, name string) error

	// StartService starts a stopped service.
	StartService(ctx context.Context, name string) error

	// StopService stops a running service.
	StopService(ctx context.Context, name string) error

	// ServiceLogs returns a reader for service logs.
	ServiceLogs(ctx context.Context, name string, tail int) (io.ReadCloser, error)

	// Down tears down the entire stack.
	Down(ctx context.Context, removeVolumes bool) error

	// Close releases any held resources.
	Close() error
}
