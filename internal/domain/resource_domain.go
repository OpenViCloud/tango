package domain

import (
	"context"
	"errors"
	"time"
)

var ErrResourceDomainNotFound = errors.New("resource domain not found")
var ErrResourceDomainConflict = errors.New("domain already in use")

type ResourceDomainType = string

const (
	ResourceDomainTypeAuto   ResourceDomainType = "auto"
	ResourceDomainTypeCustom ResourceDomainType = "custom"
)

type ResourceDomain struct {
	ID         string
	ResourceID string
	Host       string
	TargetPort int
	TLSEnabled bool
	Type       ResourceDomainType
	Verified   bool
	VerifiedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type ResourceDomainRepository interface {
	Create(ctx context.Context, domain ResourceDomain) (*ResourceDomain, error)
	ListByResource(ctx context.Context, resourceID string) ([]*ResourceDomain, error)
	GetByID(ctx context.Context, id string) (*ResourceDomain, error)
	GetByHost(ctx context.Context, host string) (*ResourceDomain, error)
	Update(ctx context.Context, domain ResourceDomain) (*ResourceDomain, error)
	SetVerified(ctx context.Context, id string, verifiedAt time.Time) error
	Delete(ctx context.Context, id string) error
	DeleteByResource(ctx context.Context, resourceID string) error
}

// TraefikFileProvider writes and removes Traefik dynamic-configuration YAML files
// so that routing changes take effect immediately without restarting containers.
type TraefikFileProvider interface {
	// Write generates a config file for the resource. containerName is the Docker
	// container name used as the backend URL (resolved via Docker DNS on tango_net).
	// TLS and target port are configured per-domain via each ResourceDomain.
	Write(resourceID string, domains []*ResourceDomain, containerName string, certResolver string) error
	// Delete removes the config file for the resource.
	Delete(resourceID string) error
	// WriteAppConfig generates the Traefik config for the Tango app itself.
	// backendURL is the app's internal URL, e.g. "http://app:8080".
	WriteAppConfig(appDomain string, tlsEnabled bool, certResolver string, backendURL string) error
	// DeleteAppConfig removes the Tango app's Traefik config file.
	DeleteAppConfig() error
	// WriteStaticConfig writes the Traefik static configuration file (traefik.yml).
	// acmeEmail enables Let's Encrypt when non-empty; empty disables ACME.
	WriteStaticConfig(acmeEmail string) error
}
