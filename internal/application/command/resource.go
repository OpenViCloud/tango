package command

import (
	"context"
	"fmt"

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
	Ports         []ResourcePortInput
	EnvVars       []ResourceEnvVarInput
}

type CreateResourceHandler struct {
	resourceRepo domain.ResourceRepository
	dockerRepo   domain.DockerRepository
}

func NewCreateResourceHandler(resourceRepo domain.ResourceRepository, dockerRepo domain.DockerRepository) *CreateResourceHandler {
	return &CreateResourceHandler{resourceRepo: resourceRepo, dockerRepo: dockerRepo}
}

func (h *CreateResourceHandler) Handle(ctx context.Context, cmd CreateResourceCommand) (*domain.Resource, error) {
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
	ID    string
	Name  string
	Ports []ResourcePortInput
}

type UpdateResourceHandler struct {
	resourceRepo domain.ResourceRepository
}

func NewUpdateResourceHandler(resourceRepo domain.ResourceRepository) *UpdateResourceHandler {
	return &UpdateResourceHandler{resourceRepo: resourceRepo}
}

func (h *UpdateResourceHandler) Handle(ctx context.Context, cmd UpdateResourceCommand) (*domain.Resource, error) {
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
		Name:  cmd.Name,
		Ports: ports,
	})
}

// ── Start Resource ────────────────────────────────────────────────────────────

type StartResourceCommand struct {
	ID string
}

type StartResourceHandler struct {
	resourceRepo domain.ResourceRepository
	dockerRepo   domain.DockerRepository
}

func NewStartResourceHandler(resourceRepo domain.ResourceRepository, dockerRepo domain.DockerRepository) *StartResourceHandler {
	return &StartResourceHandler{resourceRepo: resourceRepo, dockerRepo: dockerRepo}
}

func (h *StartResourceHandler) Handle(ctx context.Context, cmd StartResourceCommand) error {
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

		var volumes []string
		if v, ok := resource.Config["volumes"]; ok {
			if raw, ok := v.([]interface{}); ok {
				for _, item := range raw {
					if s, ok := item.(string); ok {
						volumes = append(volumes, s)
					}
				}
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

		ct, err := h.dockerRepo.CreateContainer(ctx, domain.CreateContainerInput{
			Name:         resource.Name,
			Image:        imageRef,
			Env:          env,
			PortBindings: portBindings,
			Volumes:      volumes,
			Cmd:          cmd,
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
	return h.resourceRepo.UpdateStatus(ctx, resource.ID, domain.ResourceStatusRunning, containerID)
}

// ── Stop Resource ─────────────────────────────────────────────────────────────

type StopResourceCommand struct {
	ID string
}

type StopResourceHandler struct {
	resourceRepo domain.ResourceRepository
	dockerRepo   domain.DockerRepository
}

func NewStopResourceHandler(resourceRepo domain.ResourceRepository, dockerRepo domain.DockerRepository) *StopResourceHandler {
	return &StopResourceHandler{resourceRepo: resourceRepo, dockerRepo: dockerRepo}
}

func (h *StopResourceHandler) Handle(ctx context.Context, cmd StopResourceCommand) error {
	resource, err := h.resourceRepo.GetByID(ctx, cmd.ID)
	if err != nil {
		return err
	}
	if resource.ContainerID == "" {
		// Already stopped / never started — just ensure status is correct.
		return h.resourceRepo.UpdateStatus(ctx, resource.ID, domain.ResourceStatusStopped, "")
	}
	if err := h.dockerRepo.StopContainer(ctx, resource.ContainerID); err != nil {
		return fmt.Errorf("stop container: %w", err)
	}
	return h.resourceRepo.UpdateStatus(ctx, resource.ID, domain.ResourceStatusStopped, resource.ContainerID)
}

// ── Delete Resource ───────────────────────────────────────────────────────────

type DeleteResourceCommand struct {
	ID string
}

type DeleteResourceHandler struct {
	resourceRepo domain.ResourceRepository
	dockerRepo   domain.DockerRepository
}

func NewDeleteResourceHandler(resourceRepo domain.ResourceRepository, dockerRepo domain.DockerRepository) *DeleteResourceHandler {
	return &DeleteResourceHandler{resourceRepo: resourceRepo, dockerRepo: dockerRepo}
}

func (h *DeleteResourceHandler) Handle(ctx context.Context, cmd DeleteResourceCommand) error {
	resource, err := h.resourceRepo.GetByID(ctx, cmd.ID)
	if err != nil {
		return err
	}
	if resource.ContainerID != "" {
		_ = h.dockerRepo.StopContainer(ctx, resource.ContainerID)
		_ = h.dockerRepo.RemoveContainer(ctx, resource.ContainerID, true)
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
