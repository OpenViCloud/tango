package domain

import (
	"context"
	"errors"
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
	AutoRemove   bool
}

// DockerRepository abstracts all Docker Engine operations.
type DockerRepository interface {
	ListImages(ctx context.Context) ([]Image, error)
	PullImage(ctx context.Context, input PullImageInput) error
	RemoveImage(ctx context.Context, imageID string, force bool) error
	ListContainers(ctx context.Context, all bool) ([]Container, error)
	CreateContainer(ctx context.Context, input CreateContainerInput) (Container, error)
	StartContainer(ctx context.Context, containerID string) error
	StopContainer(ctx context.Context, containerID string) error
	RemoveContainer(ctx context.Context, containerID string, force bool) error
	Close() error
}
