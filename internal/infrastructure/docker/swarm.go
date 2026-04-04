package docker

import (
	"context"
	"fmt"
	"strings"

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

	var mounts []swarm.Mount
	for _, bind := range input.Volumes {
		parts := strings.SplitN(bind, ":", 2)
		if len(parts) != 2 {
			continue
		}
		mounts = append(mounts, swarm.Mount{
			Type:   swarm.MountTypeBind,
			Source: parts[0],
			Target: parts[1],
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

	spec := swarm.ServiceSpec{
		Annotations: swarm.Annotations{Name: input.Name},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image:  input.Image,
				Env:    envSlice,
				Mounts: mounts,
			},
		},
		Networks: netAttachments,
	}

	if len(input.Cmd) > 0 {
		spec.TaskTemplate.ContainerSpec.Command = input.Cmd
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

var _ domain.SwarmRepository = (*SwarmRepository)(nil)
