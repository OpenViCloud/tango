package traefik

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// Restarter restarts the Traefik container via the Docker API so that
// updated static configuration (traefik.yml) takes effect.
type Restarter struct {
	containerName string
	client        *client.Client
}

// NewRestarter creates a Restarter that targets the named container.
// It uses the standard Docker environment variables (DOCKER_HOST, etc.).
func NewRestarter(containerName string) (*Restarter, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("create docker client for traefik restarter: %w", err)
	}
	return &Restarter{containerName: containerName, client: cli}, nil
}

// RestartTraefik restarts the Traefik container with a 10-second graceful stop timeout.
func (r *Restarter) RestartTraefik(ctx context.Context) error {
	timeout := 10
	if err := r.client.ContainerRestart(ctx, r.containerName, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("restart traefik container %q: %w", r.containerName, err)
	}
	return nil
}

// Close releases the underlying Docker client connection.
func (r *Restarter) Close() error {
	return r.client.Close()
}
