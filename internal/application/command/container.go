package command

import (
	"context"

	"tango/internal/domain"
)

// ── Pull Image ────────────────────────────────────────────────────────────────

type PullImageCommand struct {
	Reference string
}

type PullImageHandler struct {
	docker domain.DockerRepository
}

func NewPullImageHandler(docker domain.DockerRepository) *PullImageHandler {
	return &PullImageHandler{docker: docker}
}

func (h *PullImageHandler) Handle(ctx context.Context, cmd PullImageCommand) error {
	return h.docker.PullImage(ctx, domain.PullImageInput{Reference: cmd.Reference})
}

// ── Remove Image ──────────────────────────────────────────────────────────────

type RemoveImageCommand struct {
	ImageID string
	Force   bool
}

type RemoveImageHandler struct {
	docker domain.DockerRepository
}

func NewRemoveImageHandler(docker domain.DockerRepository) *RemoveImageHandler {
	return &RemoveImageHandler{docker: docker}
}

func (h *RemoveImageHandler) Handle(ctx context.Context, cmd RemoveImageCommand) error {
	return h.docker.RemoveImage(ctx, cmd.ImageID, cmd.Force)
}

// ── Create Container ──────────────────────────────────────────────────────────

type CreateContainerCommand struct {
	Name         string
	Image        string
	Cmd          []string
	Env          map[string]string
	PortBindings map[string]string // containerPort -> hostPort
	Volumes      []string          // host:container bind mounts
	AutoRemove   bool
}

type CreateContainerHandler struct {
	docker domain.DockerRepository
}

func NewCreateContainerHandler(docker domain.DockerRepository) *CreateContainerHandler {
	return &CreateContainerHandler{docker: docker}
}

func (h *CreateContainerHandler) Handle(ctx context.Context, cmd CreateContainerCommand) (domain.Container, error) {
	return h.docker.CreateContainer(ctx, domain.CreateContainerInput{
		Name:         cmd.Name,
		Image:        cmd.Image,
		Cmd:          cmd.Cmd,
		Env:          cmd.Env,
		PortBindings: cmd.PortBindings,
		Volumes:      cmd.Volumes,
		AutoRemove:   cmd.AutoRemove,
	})
}

// ── Start Container ───────────────────────────────────────────────────────────

type StartContainerCommand struct {
	ContainerID string
}

type StartContainerHandler struct {
	docker domain.DockerRepository
}

func NewStartContainerHandler(docker domain.DockerRepository) *StartContainerHandler {
	return &StartContainerHandler{docker: docker}
}

func (h *StartContainerHandler) Handle(ctx context.Context, cmd StartContainerCommand) error {
	return h.docker.StartContainer(ctx, cmd.ContainerID)
}

// ── Stop Container ────────────────────────────────────────────────────────────

type StopContainerCommand struct {
	ContainerID string
}

type StopContainerHandler struct {
	docker domain.DockerRepository
}

func NewStopContainerHandler(docker domain.DockerRepository) *StopContainerHandler {
	return &StopContainerHandler{docker: docker}
}

func (h *StopContainerHandler) Handle(ctx context.Context, cmd StopContainerCommand) error {
	return h.docker.StopContainer(ctx, cmd.ContainerID)
}

// ── Remove Container ──────────────────────────────────────────────────────────

type RemoveContainerCommand struct {
	ContainerID string
	Force       bool
}

type RemoveContainerHandler struct {
	docker domain.DockerRepository
}

func NewRemoveContainerHandler(docker domain.DockerRepository) *RemoveContainerHandler {
	return &RemoveContainerHandler{docker: docker}
}

func (h *RemoveContainerHandler) Handle(ctx context.Context, cmd RemoveContainerCommand) error {
	return h.docker.RemoveContainer(ctx, cmd.ContainerID, cmd.Force)
}
