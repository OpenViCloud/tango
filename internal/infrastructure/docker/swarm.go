package docker

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
	"tango/internal/domain"
)

// SwarmRepository implements domain.SwarmRepository using the Docker Engine API.
// It is separate from Repository so it can be nil-checked independently when
// the node is not a swarm manager.
type SwarmRepository struct {
	client *Repository // reuse the underlying Docker client
}

// NewSwarmRepository wraps an existing docker Repository for swarm operations.
func NewSwarmRepository(r *Repository) *SwarmRepository {
	return &SwarmRepository{client: r}
}

// IsManager returns true when the local Docker daemon is an active swarm manager.
func (s *SwarmRepository) IsManager(ctx context.Context) bool {
	info, err := s.client.client.Info(ctx)
	if err != nil {
		return false
	}
	return info.Swarm.LocalNodeState == "active" && info.Swarm.ControlAvailable
}

// EnsureOverlayNetwork creates an overlay network if it does not already exist.
func (s *SwarmRepository) EnsureOverlayNetwork(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	if _, err := s.client.client.NetworkInspect(ctx, name, network.InspectOptions{}); err == nil {
		return nil
	}
	_, err := s.client.client.NetworkCreate(ctx, name, network.CreateOptions{
		Driver:     "overlay",
		Attachable: true,
	})
	if err != nil {
		// Tolerate a race where another process created it first.
		if _, inspectErr := s.client.client.NetworkInspect(ctx, name, network.InspectOptions{}); inspectErr == nil {
			return nil
		}
		return fmt.Errorf("ensure overlay network %s: %w", name, err)
	}
	return nil
}

// CreateService creates a Docker Swarm service and returns its ID and name.
func (s *SwarmRepository) CreateService(ctx context.Context, input domain.CreateServiceInput) (domain.SwarmService, error) {
	envSlice := envMapToSlice(input.Env)

	var mounts []mount.Mount
	for _, bind := range input.Volumes {
		parts := strings.SplitN(bind, ":", 2)
		if len(parts) != 2 {
			continue
		}
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: parts[0],
			Target: parts[1],
			BindOptions: &mount.BindOptions{
				CreateMountpoint: true,
			},
		})
	}

	// Attach to each named overlay network.
	var netAttachments []swarm.NetworkAttachmentConfig
	for _, netName := range input.Networks {
		if err := s.EnsureOverlayNetwork(ctx, netName); err != nil {
			return domain.SwarmService{}, err
		}
		netAttachments = append(netAttachments, swarm.NetworkAttachmentConfig{Target: netName})
	}

	replicas := input.Replicas
	if replicas == 0 {
		replicas = 1
	}

	restartCondition := swarm.RestartPolicyConditionAny
	spec := swarm.ServiceSpec{
		Annotations: swarm.Annotations{Name: input.Name},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image:  input.Image,
				Env:    envSlice,
				Mounts: mounts,
			},
			RestartPolicy: &swarm.RestartPolicy{
				Condition: restartCondition,
			},
		},
		Networks: netAttachments,
		Mode: swarm.ServiceMode{
			Replicated: &swarm.ReplicatedService{Replicas: &replicas},
		},
	}

	if len(input.Cmd) > 0 {
		spec.TaskTemplate.ContainerSpec.Command = input.Cmd
	}

	// Resource limits: apply when non-zero.
	if input.MemoryLimit > 0 || input.CPULimit > 0 {
		res := &swarm.ResourceRequirements{}
		if input.MemoryLimit > 0 || input.CPULimit > 0 {
			res.Limits = &swarm.Limit{
				MemoryBytes: input.MemoryLimit,
				NanoCPUs:    input.CPULimit,
			}
		}
		spec.TaskTemplate.Resources = res
	}

	// Placement constraint: pin to a specific node when NodeID is set.
	if input.NodeID != "" {
		spec.TaskTemplate.Placement = &swarm.Placement{
			Constraints: []string{fmt.Sprintf("node.id == %s", input.NodeID)},
		}
	}

	resp, err := s.client.client.ServiceCreate(ctx, spec, swarm.ServiceCreateOptions{})
	if err != nil {
		return domain.SwarmService{}, fmt.Errorf("create swarm service %s: %w", input.Name, err)
	}

	return domain.SwarmService{
		ID:   resp.ID,
		Name: input.Name,
	}, nil
}

// ScaleService updates the replica count of an existing swarm service.
func (s *SwarmRepository) ScaleService(ctx context.Context, serviceID string, replicas uint64) error {
	svc, _, err := s.client.client.ServiceInspectWithRaw(ctx, serviceID, swarm.ServiceInspectOptions{})
	if err != nil {
		return fmt.Errorf("inspect swarm service %s: %w", serviceID, err)
	}
	if replicas == 0 {
		replicas = 1
	}
	svc.Spec.Mode = swarm.ServiceMode{
		Replicated: &swarm.ReplicatedService{Replicas: &replicas},
	}
	_, err = s.client.client.ServiceUpdate(ctx, serviceID, svc.Version, svc.Spec, swarm.ServiceUpdateOptions{})
	if err != nil {
		return fmt.Errorf("scale swarm service %s to %d: %w", serviceID, replicas, err)
	}
	return nil
}

// RemoveService removes a swarm service and all its tasks.
func (s *SwarmRepository) RemoveService(ctx context.Context, serviceID string) error {
	if err := s.client.client.ServiceRemove(ctx, serviceID); err != nil {
		return fmt.Errorf("remove swarm service %s: %w", serviceID, err)
	}
	return nil
}

// ListNodes returns all nodes registered in the swarm.
func (s *SwarmRepository) ListNodes(ctx context.Context) ([]domain.SwarmNode, error) {
	nodes, err := s.client.client.NodeList(ctx, swarm.NodeListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list swarm nodes: %w", err)
	}

	result := make([]domain.SwarmNode, 0, len(nodes))
	for _, n := range nodes {
		role := string(n.Spec.Role)
		managerAddr := ""
		if n.ManagerStatus != nil {
			managerAddr = n.ManagerStatus.Addr
		}
		result = append(result, domain.SwarmNode{
			ID:           n.ID,
			Hostname:     n.Description.Hostname,
			Role:         role,
			State:        string(n.Status.State),
			Availability: string(n.Spec.Availability),
			ManagerAddr:  managerAddr,
		})
	}
	return result, nil
}

// ServiceRunning reports whether the service exists and has at least one running task.
// Returns false (not an error) when the service is not found.
func (s *SwarmRepository) ServiceRunning(ctx context.Context, serviceID string) (bool, error) {
	if strings.TrimSpace(serviceID) == "" {
		return false, nil
	}
	_, _, err := s.client.client.ServiceInspectWithRaw(ctx, serviceID, swarm.ServiceInspectOptions{})
	if err != nil {
		if isNotFoundError(err) {
			return false, nil
		}
		return false, fmt.Errorf("inspect swarm service %s: %w", serviceID, err)
	}
	// Check if at least one task is running.
	tasks, err := s.client.client.TaskList(ctx, swarm.TaskListOptions{
		Filters: filters.NewArgs(
			filters.Arg("service", serviceID),
			filters.Arg("desired-state", "running"),
		),
	})
	if err != nil {
		return false, fmt.Errorf("list tasks for service %s: %w", serviceID, err)
	}
	for _, t := range tasks {
		if t.Status.State == swarm.TaskStateRunning {
			return true, nil
		}
	}
	// Service exists but no running task yet (e.g. still starting).
	return true, nil
}

// GetServiceLogs returns the most recent log lines for a swarm service.
func (s *SwarmRepository) GetServiceLogs(ctx context.Context, serviceID string, tail string) ([]string, error) {
	if tail == "" {
		tail = "200"
	}
	reader, err := s.client.client.ServiceLogs(ctx, serviceID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: false,
		Tail:       tail,
	})
	if err != nil {
		return nil, fmt.Errorf("get service logs %s: %w", serviceID, err)
	}
	defer reader.Close()

	// Swarm service logs use the same multiplexed stream format as container logs.
	var out []string
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		// Strip the 8-byte docker stream header if present.
		if len(line) > 8 {
			line = line[8:]
		}
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read service logs %s: %w", serviceID, err)
	}
	if out == nil {
		return []string{}, nil
	}
	return out, nil
}

// isNotFoundError returns true when the Docker API signals a 404 / "no such" error.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such") || strings.Contains(msg, "not found") || strings.Contains(msg, "404")
}

var _ domain.SwarmRepository = (*SwarmRepository)(nil)
