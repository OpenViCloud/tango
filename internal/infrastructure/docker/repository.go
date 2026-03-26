package docker

import (
	"context"
	"fmt"
	"io"
	"strings"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	"tango/internal/domain"
)

// Repository wraps the Docker Engine API client.
type Repository struct {
	client *client.Client
}

// NewRepository creates a Docker client using environment variables
// (DOCKER_HOST, DOCKER_TLS_VERIFY, DOCKER_CERT_PATH, DOCKER_API_VERSION).
func NewRepository() (*Repository, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}
	return &Repository{client: cli}, nil
}

// Close releases the underlying connection.
func (r *Repository) Close() error {
	return r.client.Close()
}

// ListImages returns all local Docker images.
func (r *Repository) ListImages(ctx context.Context) ([]domain.Image, error) {
	items, err := r.client.ImageList(ctx, image.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("list images: %w", err)
	}

	result := make([]domain.Image, 0, len(items))
	for _, item := range items {
		digest := ""
		if len(item.RepoDigests) > 0 {
			digest = item.RepoDigests[0]
		}
		result = append(result, domain.Image{
			ID:      item.ID,
			Tags:    item.RepoTags,
			Size:    item.Size,
			Created: item.Created,
			Digest:  digest,
			InUse:   item.Containers,
		})
	}
	return result, nil
}

// PullImage pulls an image from a registry. It streams and discards output.
func (r *Repository) PullImage(ctx context.Context, input domain.PullImageInput) error {
	out, err := r.client.ImagePull(ctx, input.Reference, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pull image %s: %w", input.Reference, err)
	}
	defer out.Close()
	_, _ = io.Copy(io.Discard, out)
	return nil
}

// PullImageStream starts an image pull and returns the raw NDJSON event stream.
// The caller is responsible for closing the returned ReadCloser.
func (r *Repository) PullImageStream(ctx context.Context, reference string) (io.ReadCloser, error) {
	out, err := r.client.ImagePull(ctx, reference, image.PullOptions{})
	if err != nil {
		return nil, fmt.Errorf("pull image %s: %w", reference, err)
	}
	return out, nil
}

// RemoveImage removes an image by ID or tag.
func (r *Repository) RemoveImage(ctx context.Context, imageID string, force bool) error {
	_, err := r.client.ImageRemove(ctx, imageID, image.RemoveOptions{
		Force:         force,
		PruneChildren: true,
	})
	if err != nil {
		return fmt.Errorf("remove image %s: %w", imageID, err)
	}
	return nil
}

// ListContainers returns containers. Pass all=true to include stopped ones.
func (r *Repository) ListContainers(ctx context.Context, all bool) ([]domain.Container, error) {
	items, err := r.client.ContainerList(ctx, container.ListOptions{All: all})
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	result := make([]domain.Container, 0, len(items))
	for _, item := range items {
		result = append(result, mapContainerSummary(item))
	}
	return result, nil
}

// CreateContainer creates (but does not start) a new container.
func (r *Repository) CreateContainer(ctx context.Context, input domain.CreateContainerInput) (domain.Container, error) {
	portSet := nat.PortSet{}
	portMap := nat.PortMap{}

	for _, p := range input.ExposedPorts {
		port, err := nat.NewPort("tcp", p)
		if err != nil {
			return domain.Container{}, fmt.Errorf("parse exposed port %s: %w", p, err)
		}
		portSet[port] = struct{}{}
	}

	for containerPort, hostPort := range input.PortBindings {
		port, err := nat.NewPort("tcp", containerPort)
		if err != nil {
			return domain.Container{}, fmt.Errorf("parse port binding %s: %w", containerPort, err)
		}
		portSet[port] = struct{}{}
		portMap[port] = []nat.PortBinding{{HostPort: hostPort}}
	}

	cfg := &container.Config{
		Image:        input.Image,
		Cmd:          input.Cmd,
		Tty:          input.TTY,
		OpenStdin:    input.OpenStdin,
		Env:          envMapToSlice(input.Env),
		ExposedPorts: portSet,
	}

	hostCfg := &container.HostConfig{
		AutoRemove:   input.AutoRemove,
		PortBindings: portMap,
	}

	resp, err := r.client.ContainerCreate(ctx, cfg, hostCfg, nil, nil, input.Name)
	if err != nil {
		return domain.Container{}, fmt.Errorf("create container: %w", err)
	}

	inspect, err := r.client.ContainerInspect(ctx, resp.ID)
	if err != nil {
		return domain.Container{}, fmt.Errorf("inspect container after create: %w", err)
	}
	return mapInspect(inspect), nil
}

// StartContainer starts a stopped or newly created container.
func (r *Repository) StartContainer(ctx context.Context, containerID string) error {
	if err := r.client.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("start container %s: %w", containerID, err)
	}
	return nil
}

// StopContainer sends a SIGTERM to the container and waits for it to exit.
func (r *Repository) StopContainer(ctx context.Context, containerID string) error {
	if err := r.client.ContainerStop(ctx, containerID, container.StopOptions{}); err != nil {
		return fmt.Errorf("stop container %s: %w", containerID, err)
	}
	return nil
}

// RemoveContainer removes a container. Pass force=true to remove running containers.
func (r *Repository) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	f := filters.NewArgs(filters.Arg("id", containerID))
	_ = f
	if err := r.client.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force:         force,
		RemoveVolumes: true,
	}); err != nil {
		return fmt.Errorf("remove container %s: %w", containerID, err)
	}
	return nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func mapContainerSummary(item dockertypes.Container) domain.Container {
	name := ""
	if len(item.Names) > 0 {
		name = strings.TrimPrefix(item.Names[0], "/")
	}

	ports := make([]domain.ContainerPort, 0, len(item.Ports))
	for _, p := range item.Ports {
		ports = append(ports, domain.ContainerPort{
			IP:          p.IP,
			PrivatePort: p.PrivatePort,
			PublicPort:  p.PublicPort,
			Type:        p.Type,
		})
	}

	return domain.Container{
		ID:      item.ID,
		Name:    name,
		Image:   item.Image,
		ImageID: item.ImageID,
		State:   item.State,
		Status:  item.Status,
		Command: item.Command,
		Ports:   ports,
		Labels:  item.Labels,
	}
}

func mapInspect(item dockertypes.ContainerJSON) domain.Container {
	name := strings.TrimPrefix(item.Name, "/")

	ports := make([]domain.ContainerPort, 0)
	if item.NetworkSettings != nil {
		for port, bindings := range item.NetworkSettings.Ports {
			for _, b := range bindings {
				pub := uint16(0)
				fmt.Sscanf(b.HostPort, "%d", &pub)
				ports = append(ports, domain.ContainerPort{
					IP:          b.HostIP,
					PrivatePort: uint16(port.Int()),
					PublicPort:  pub,
					Type:        port.Proto(),
				})
			}
		}
	}

	state := ""
	status := ""
	if item.State != nil {
		state = item.State.Status
		if item.State.Running {
			status = "running"
		} else {
			status = "exited"
		}
	}

	cmd := ""
	if item.Config != nil {
		cmd = strings.Join(item.Config.Cmd, " ")
	}

	img := ""
	imgID := ""
	if item.Config != nil {
		img = item.Config.Image
	}
	imgID = item.Image

	return domain.Container{
		ID:      item.ID,
		Name:    name,
		Image:   img,
		ImageID: imgID,
		State:   state,
		Status:  status,
		Command: cmd,
		Ports:   ports,
		Labels:  item.Config.Labels,
	}
}

func envMapToSlice(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for k, v := range values {
		result = append(result, k+"="+v)
	}
	return result
}
