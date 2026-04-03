package domain

import (
	"context"
	"errors"
	"time"
)

var ErrPlatformConfigNotFound = errors.New("platform config not found")

const (
	PlatformConfigPublicIP          = "public_ip"
	PlatformConfigBaseDomain        = "base_domain"
	PlatformConfigWildcardEnabled   = "wildcard_enabled"
	PlatformConfigTraefikNetwork    = "traefik_network"
	PlatformConfigCertResolver      = "cert_resolver"
	PlatformConfigAppDomain         = "app_domain"
	PlatformConfigAppTLSEnabled     = "app_tls_enabled"
	PlatformConfigAppBackendURL     = "app_backend_url"
	PlatformConfigResourceMountRoot = "resource_mount_root"
	PlatformConfigACMEEmail         = "acme_email"
)

type PlatformConfig struct {
	Key       string
	Value     string
	UpdatedAt time.Time
}

type PlatformConfigRepository interface {
	Get(ctx context.Context, key string) (*PlatformConfig, error)
	Set(ctx context.Context, key, value string) error
	List(ctx context.Context) ([]*PlatformConfig, error)
}
