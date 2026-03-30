package domain

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	SetVerified(ctx context.Context, id string, verifiedAt time.Time) error
	Delete(ctx context.Context, id string) error
	DeleteByResource(ctx context.Context, resourceID string) error
}

// TraefikConfig holds platform-level Traefik settings used when generating Docker labels.
type TraefikConfig struct {
	Network      string // Docker network Traefik listens on (traefik.docker.network)
	TLSEnabled   bool   // When true, generate HTTPS router + HTTP→HTTPS redirect
	CertResolver string // Let's Encrypt resolver name, e.g. "letsencrypt"
}

// TraefikLabels generates Traefik Docker labels for the given domains and internal port.
// Auto domains are always served over HTTP only.
// Verified custom domains are served over HTTPS (with HTTP→HTTPS redirect) when TLS is enabled.
// Returns nil when there are no routable domains.
func TraefikLabels(resourceID string, domains []*ResourceDomain, internalPort int, cfg TraefikConfig) map[string]string {
	var autoHosts []string
	var customHosts []string
	for _, d := range domains {
		if d.Type == ResourceDomainTypeAuto {
			autoHosts = append(autoHosts, fmt.Sprintf("Host(`%s`)", d.Host))
		} else if d.Verified {
			customHosts = append(customHosts, fmt.Sprintf("Host(`%s`)", d.Host))
		}
	}
	if len(autoHosts) == 0 && len(customHosts) == 0 {
		return nil
	}

	// Use first 12 chars of resource ID as router name (safe for Traefik)
	routerName := "r-" + strings.ReplaceAll(resourceID, "-", "")[:12]
	svcName := routerName + "-svc"
	port := fmt.Sprintf("%d", internalPort)

	labels := map[string]string{
		"traefik.enable": "true",
		fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", svcName): port,
	}

	if cfg.Network != "" {
		labels["traefik.docker.network"] = cfg.Network
	}

	// Auto domains: always HTTP only (localhost / internal hostnames cannot get TLS certs)
	if len(autoHosts) > 0 {
		autoRouter := routerName + "-auto"
		labels[fmt.Sprintf("traefik.http.routers.%s.rule", autoRouter)] = rule(autoHosts)
		labels[fmt.Sprintf("traefik.http.routers.%s.entrypoints", autoRouter)] = "web"
		labels[fmt.Sprintf("traefik.http.routers.%s.service", autoRouter)] = svcName
	}

	// Custom domains: HTTPS when TLS enabled, HTTP otherwise
	if len(customHosts) > 0 {
		customRule := rule(customHosts)
		if cfg.TLSEnabled && cfg.CertResolver != "" {
			// HTTP → HTTPS redirect
			httpRouter := routerName + "-http"
			mw := routerName + "-redirect"
			labels[fmt.Sprintf("traefik.http.routers.%s.rule", httpRouter)] = customRule
			labels[fmt.Sprintf("traefik.http.routers.%s.entrypoints", httpRouter)] = "web"
			labels[fmt.Sprintf("traefik.http.routers.%s.middlewares", httpRouter)] = mw
			labels[fmt.Sprintf("traefik.http.middlewares.%s.redirectscheme.scheme", mw)] = "https"
			labels[fmt.Sprintf("traefik.http.middlewares.%s.redirectscheme.permanent", mw)] = "true"

			// HTTPS router — only custom verified domains get certresolver
			labels[fmt.Sprintf("traefik.http.routers.%s.rule", routerName)] = customRule
			labels[fmt.Sprintf("traefik.http.routers.%s.entrypoints", routerName)] = "websecure"
			labels[fmt.Sprintf("traefik.http.routers.%s.service", routerName)] = svcName
			labels[fmt.Sprintf("traefik.http.routers.%s.tls", routerName)] = "true"
			labels[fmt.Sprintf("traefik.http.routers.%s.tls.certresolver", routerName)] = cfg.CertResolver
		} else {
			// HTTP only for custom domains
			labels[fmt.Sprintf("traefik.http.routers.%s.rule", routerName)] = customRule
			labels[fmt.Sprintf("traefik.http.routers.%s.entrypoints", routerName)] = "web"
			labels[fmt.Sprintf("traefik.http.routers.%s.service", routerName)] = svcName
		}
	}

	return labels
}

func rule(hosts []string) string {
	return strings.Join(hosts, " || ")
}
