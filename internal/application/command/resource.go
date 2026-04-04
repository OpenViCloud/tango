package command

import (
	"context"
	"fmt"
	"os"
	"strings"

	"tango/internal/config"
	"tango/internal/domain"
)

func buildResourceRuntime(cmd struct {
	Name    string
	Image   string
	Tag     string
	Ports   []ResourcePortInput
	EnvVars []ResourceEnvVarInput
}) (string, map[string]string, map[string]string) {
	imageRef := cmd.Image
	if cmd.Tag != "" {
		imageRef = cmd.Image + ":" + cmd.Tag
	}

	portBindings := make(map[string]string)
	for _, p := range cmd.Ports {
		proto := p.Proto
		if proto == "" {
			proto = "tcp"
		}
		containerPort := fmt.Sprintf("%d/%s", p.InternalPort, proto)
		portBindings[containerPort] = fmt.Sprintf("%d", p.HostPort)
	}

	env := make(map[string]string)
	for _, ev := range cmd.EnvVars {
		env[ev.Key] = ev.Value
	}

	return imageRef, portBindings, env
}

// ── Create Resource ───────────────────────────────────────────────────────────

type ResourcePortInput struct {
	HostPort     int
	InternalPort int
	Proto        string
	Label        string
}

type ResourceEnvVarInput struct {
	Key      string
	Value    string
	IsSecret bool
}

type CreateResourceCommand struct {
	ID            string
	Name          string
	Type          domain.ResourceType
	Image         string
	Tag           string
	Config        map[string]any
	EnvironmentID string
	CreatedBy     string
	TLSEnabled    bool
	Ports         []ResourcePortInput
	EnvVars       []ResourceEnvVarInput
}

type CreateResourceHandler struct {
	resourceRepo   domain.ResourceRepository
	dockerRepo     domain.DockerRepository
	domainRepo     domain.ResourceDomainRepository
	platformConfig domain.PlatformConfigRepository
}

func NewCreateResourceHandler(
	resourceRepo domain.ResourceRepository,
	dockerRepo domain.DockerRepository,
	domainRepo domain.ResourceDomainRepository,
	platformConfig domain.PlatformConfigRepository,
) *CreateResourceHandler {
	return &CreateResourceHandler{
		resourceRepo:   resourceRepo,
		dockerRepo:     dockerRepo,
		domainRepo:     domainRepo,
		platformConfig: platformConfig,
	}
}

func (h *CreateResourceHandler) Handle(ctx context.Context, cmd CreateResourceCommand) (*domain.Resource, error) {
	mountRoot := resolveResourceMountRoot(ctx, h.platformConfig)
	mounts, err := domain.ResolveResourceMounts(cmd.Config, mountRoot)
	if err != nil {
		return nil, err
	}
	for _, hostPath := range mounts.HostPaths {
		if err := os.MkdirAll(hostPath, 0o755); err != nil {
			return nil, fmt.Errorf("prepare resource volume %s: %w", hostPath, err)
		}
	}

	ports := make([]domain.ResourcePort, 0, len(cmd.Ports))
	for _, p := range cmd.Ports {
		proto := p.Proto
		if proto == "" {
			proto = "tcp"
		}
		ports = append(ports, domain.ResourcePort{
			HostPort:     p.HostPort,
			InternalPort: p.InternalPort,
			Proto:        proto,
			Label:        p.Label,
		})
	}

	envVars := make([]domain.ResourceEnvVar, 0, len(cmd.EnvVars))
	for _, ev := range cmd.EnvVars {
		envVars = append(envVars, domain.ResourceEnvVar{
			Key:      ev.Key,
			Value:    ev.Value,
			IsSecret: ev.IsSecret,
		})
	}

	resource, dbErr := h.resourceRepo.Create(ctx, domain.CreateResourceInput{
		ID:            cmd.ID,
		Name:          cmd.Name,
		Type:          cmd.Type,
		Image:         cmd.Image,
		Tag:           cmd.Tag,
		Config:        cmd.Config,
		EnvironmentID: cmd.EnvironmentID,
		CreatedBy:     cmd.CreatedBy,
		TLSEnabled:    cmd.TLSEnabled,
		Ports:         ports,
		EnvVars:       envVars,
	})
	if dbErr != nil {
		return nil, fmt.Errorf("save resource: %w", dbErr)
	}

	if err := h.resourceRepo.UpdateStatus(ctx, resource.ID, domain.ResourceStatusStopped, ""); err != nil {
		return nil, fmt.Errorf("update resource status: %w", err)
	}

	return h.resourceRepo.GetByID(ctx, resource.ID)
}

// ── Update Resource ───────────────────────────────────────────────────────────

type UpdateResourceCommand struct {
	ID         string
	Name       string
	TLSEnabled bool
	Ports      []ResourcePortInput
	Config     map[string]any
}

type UpdateResourceHandler struct {
	resourceRepo   domain.ResourceRepository
	platformConfig domain.PlatformConfigRepository
}

func NewUpdateResourceHandler(resourceRepo domain.ResourceRepository, platformConfig domain.PlatformConfigRepository) *UpdateResourceHandler {
	return &UpdateResourceHandler{resourceRepo: resourceRepo, platformConfig: platformConfig}
}

func (h *UpdateResourceHandler) Handle(ctx context.Context, cmd UpdateResourceCommand) (*domain.Resource, error) {
	configToSave := cmd.Config
	if configToSave == nil {
		resource, err := h.resourceRepo.GetByID(ctx, cmd.ID)
		if err != nil {
			return nil, err
		}
		configToSave = resource.Config
	}
	if configToSave != nil {
		mountRoot := resolveResourceMountRoot(ctx, h.platformConfig)
		mounts, err := domain.ResolveResourceMounts(configToSave, mountRoot)
		if err != nil {
			return nil, err
		}
		for _, hostPath := range mounts.HostPaths {
			if err := os.MkdirAll(hostPath, 0o755); err != nil {
				return nil, fmt.Errorf("prepare resource volume %s: %w", hostPath, err)
			}
		}
	}
	ports := make([]domain.ResourcePort, 0, len(cmd.Ports))
	for _, p := range cmd.Ports {
		proto := p.Proto
		if proto == "" {
			proto = "tcp"
		}
		ports = append(ports, domain.ResourcePort{
			HostPort:     p.HostPort,
			InternalPort: p.InternalPort,
			Proto:        proto,
			Label:        p.Label,
		})
	}
	return h.resourceRepo.Update(ctx, cmd.ID, domain.UpdateResourceInput{
		Name:       cmd.Name,
		TLSEnabled: cmd.TLSEnabled,
		Ports:      ports,
		Config:     configToSave,
	})
}

// ── Start Resource ────────────────────────────────────────────────────────────

type StartResourceCommand struct {
	ID string
}

type StartResourceHandler struct {
	resourceRepo   domain.ResourceRepository
	dockerRepo     domain.DockerRepository
	domainRepo     domain.ResourceDomainRepository
	platformConfig domain.PlatformConfigRepository
	fileProvider   domain.TraefikFileProvider
}

func NewStartResourceHandler(
	resourceRepo domain.ResourceRepository,
	dockerRepo domain.DockerRepository,
	domainRepo domain.ResourceDomainRepository,
	platformConfig domain.PlatformConfigRepository,
	fileProvider domain.TraefikFileProvider,
) *StartResourceHandler {
	return &StartResourceHandler{resourceRepo: resourceRepo, dockerRepo: dockerRepo, domainRepo: domainRepo, platformConfig: platformConfig, fileProvider: fileProvider}
}

func (h *StartResourceHandler) Handle(ctx context.Context, cmd StartResourceCommand) error {
	const defaultTraefikNetwork = "tango_net"

	resource, err := h.resourceRepo.GetByID(ctx, cmd.ID)
	if err != nil {
		return err
	}

	containerID := resource.ContainerID
	if containerID == "" {
		portInputs := make([]ResourcePortInput, 0, len(resource.Ports))
		for _, p := range resource.Ports {
			portInputs = append(portInputs, ResourcePortInput{
				HostPort:     p.HostPort,
				InternalPort: p.InternalPort,
				Proto:        p.Proto,
				Label:        p.Label,
			})
		}

		envInputs := make([]ResourceEnvVarInput, 0, len(resource.EnvVars))
		for _, ev := range resource.EnvVars {
			envInputs = append(envInputs, ResourceEnvVarInput{
				Key:      ev.Key,
				Value:    ev.Value,
				IsSecret: ev.IsSecret,
			})
		}

		imageRef, portBindings, env := buildResourceRuntime(struct {
			Name    string
			Image   string
			Tag     string
			Ports   []ResourcePortInput
			EnvVars []ResourceEnvVarInput
		}{
			Name:    resource.Name,
			Image:   resource.Image,
			Tag:     resource.Tag,
			Ports:   portInputs,
			EnvVars: envInputs,
		})

		mountRoot := resolveResourceMountRoot(ctx, h.platformConfig)
		mounts, err := domain.ResolveResourceMounts(resource.Config, mountRoot)
		if err != nil {
			return err
		}
		for _, hostPath := range mounts.HostPaths {
			if err := os.MkdirAll(hostPath, 0o755); err != nil {
				return fmt.Errorf("prepare resource volume %s: %w", hostPath, err)
			}
		}
		var cmd []string
		if v, ok := resource.Config["cmd"]; ok {
			if raw, ok := v.([]interface{}); ok {
				for _, item := range raw {
					if s, ok := item.(string); ok {
						cmd = append(cmd, s)
					}
				}
			}
		}

		// Resolve Traefik network from platform config
		traefikNetwork := ""
		if h.platformConfig != nil {
			if cfg, err := h.platformConfig.Get(ctx, domain.PlatformConfigTraefikNetwork); err == nil {
				traefikNetwork = cfg.Value
			}
		}
		traefikNetwork = strings.TrimSpace(traefikNetwork)
		if traefikNetwork == "" {
			traefikNetwork = defaultTraefikNetwork
		}
		networks := []string{traefikNetwork}
		if err := h.dockerRepo.EnsureNetwork(ctx, traefikNetwork); err != nil {
			return fmt.Errorf("ensure shared docker network: %w", err)
		}

		ct, err := h.dockerRepo.CreateContainer(ctx, domain.CreateContainerInput{
			Name:         resource.Name,
			Image:        imageRef,
			Env:          env,
			PortBindings: portBindings,
			Volumes:      mounts.Binds,
			Cmd:          cmd,
			Networks:     networks,
		})
		if err != nil {
			return fmt.Errorf("create container: %w", err)
		}
		containerID = ct.ID
	}

	if containerID == "" {
		return domain.ErrResourceNotStarted
	}
	if err := h.dockerRepo.StartContainer(ctx, containerID); err != nil {
		return fmt.Errorf("start container: %w", err)
	}
	if err := h.resourceRepo.UpdateStatus(ctx, resource.ID, domain.ResourceStatusRunning, containerID); err != nil {
		return err
	}

	// Write Traefik file config after container is running
	if h.fileProvider != nil && h.domainRepo != nil && h.dockerRepo != nil {
		if info, err := h.dockerRepo.InspectContainer(ctx, containerID); err == nil {
			if domains, err := h.domainRepo.ListByResource(ctx, resource.ID); err == nil && len(domains) > 0 {
				certResolver := ""
				if h.platformConfig != nil {
					if cfg, err := h.platformConfig.Get(ctx, domain.PlatformConfigCertResolver); err == nil {
						certResolver = cfg.Value
					}
				}
				_ = h.fileProvider.Write(resource.ID, domains, info.Name, certResolver)
			}
		}
	}
	return nil
}

func resolveResourceMountRoot(ctx context.Context, repo domain.PlatformConfigRepository) string {
	if repo != nil {
		if cfg, err := repo.Get(ctx, domain.PlatformConfigResourceMountRoot); err == nil {
			if value := strings.TrimSpace(cfg.Value); value != "" {
				return value
			}
		}
	}
	return config.DefaultResourceMountRootHost
}

// ── Stop Resource ─────────────────────────────────────────────────────────────

type StopResourceCommand struct {
	ID string
}

type StopResourceHandler struct {
	resourceRepo domain.ResourceRepository
	dockerRepo   domain.DockerRepository
	swarmRepo    domain.SwarmRepository
	fileProvider domain.TraefikFileProvider
}

func NewStopResourceHandler(resourceRepo domain.ResourceRepository, dockerRepo domain.DockerRepository, swarmRepo domain.SwarmRepository, fileProvider domain.TraefikFileProvider) *StopResourceHandler {
	return &StopResourceHandler{resourceRepo: resourceRepo, dockerRepo: dockerRepo, swarmRepo: swarmRepo, fileProvider: fileProvider}
}

func (h *StopResourceHandler) Handle(ctx context.Context, cmd StopResourceCommand) error {
	resource, err := h.resourceRepo.GetByID(ctx, cmd.ID)
	if err != nil {
		return err
	}
	if resource.ContainerID == "" {
		return h.resourceRepo.UpdateStatus(ctx, resource.ID, domain.ResourceStatusStopped, "")
	}

	if h.swarmRepo != nil && h.swarmRepo.IsManager(ctx) {
		if err := h.swarmRepo.RemoveService(ctx, resource.ContainerID); err != nil {
			return fmt.Errorf("remove swarm service: %w", err)
		}
	} else {
		if err := h.dockerRepo.StopContainer(ctx, resource.ContainerID); err != nil {
			return fmt.Errorf("stop container: %w", err)
		}
		_ = h.dockerRepo.RemoveContainer(ctx, resource.ContainerID, false)
	}

	if h.fileProvider != nil {
		_ = h.fileProvider.Delete(resource.ID)
	}
	return h.resourceRepo.UpdateStatus(ctx, resource.ID, domain.ResourceStatusStopped, "")
}

// ── Delete Resource ───────────────────────────────────────────────────────────

type DeleteResourceCommand struct {
	ID string
}

type DeleteResourceHandler struct {
	resourceRepo domain.ResourceRepository
	dockerRepo   domain.DockerRepository
	swarmRepo    domain.SwarmRepository
	fileProvider domain.TraefikFileProvider
}

func NewDeleteResourceHandler(resourceRepo domain.ResourceRepository, dockerRepo domain.DockerRepository, swarmRepo domain.SwarmRepository, fileProvider domain.TraefikFileProvider) *DeleteResourceHandler {
	return &DeleteResourceHandler{resourceRepo: resourceRepo, dockerRepo: dockerRepo, swarmRepo: swarmRepo, fileProvider: fileProvider}
}

func (h *DeleteResourceHandler) Handle(ctx context.Context, cmd DeleteResourceCommand) error {
	resource, err := h.resourceRepo.GetByID(ctx, cmd.ID)
	if err != nil {
		return err
	}
	if resource.ContainerID != "" {
		if h.swarmRepo != nil && h.swarmRepo.IsManager(ctx) {
			_ = h.swarmRepo.RemoveService(ctx, resource.ContainerID)
		} else {
			_ = h.dockerRepo.StopContainer(ctx, resource.ContainerID)
			_ = h.dockerRepo.RemoveContainer(ctx, resource.ContainerID, true)
		}
	}
	if h.fileProvider != nil {
		_ = h.fileProvider.Delete(resource.ID)
	}
	return h.resourceRepo.Delete(ctx, resource.ID)
}

// ── Set Resource Env Vars ─────────────────────────────────────────────────────

type SetResourceEnvVarsCommand struct {
	ResourceID string
	Vars       []ResourceEnvVarInput
}

type SetResourceEnvVarsHandler struct {
	resourceRepo domain.ResourceRepository
}

func NewSetResourceEnvVarsHandler(resourceRepo domain.ResourceRepository) *SetResourceEnvVarsHandler {
	return &SetResourceEnvVarsHandler{resourceRepo: resourceRepo}
}

func (h *SetResourceEnvVarsHandler) Handle(ctx context.Context, cmd SetResourceEnvVarsCommand) error {
	vars := make([]domain.ResourceEnvVar, 0, len(cmd.Vars))
	for _, v := range cmd.Vars {
		vars = append(vars, domain.ResourceEnvVar{
			Key:      v.Key,
			Value:    v.Value,
			IsSecret: v.IsSecret,
		})
	}
	return h.resourceRepo.SetEnvVars(ctx, cmd.ResourceID, vars)
}
