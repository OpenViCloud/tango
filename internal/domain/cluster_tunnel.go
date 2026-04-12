package domain

import (
	"context"
	"errors"
	"time"
)

var ErrClusterTunnelNotFound = errors.New("cluster tunnel not found")
var ErrClusterTunnelAlreadyExists = errors.New("cluster tunnel already exists")
var ErrClusterTunnelConnectionRequired = errors.New("cluster tunnel requires a cloudflare connection for this operation")

// TunnelExposure represents one hostname → in-cluster service mapping
// managed by a single cloudflared tunnel.
type TunnelExposure struct {
	ID         string
	Hostname   string // e.g. nginx.yourdomain.com
	ServiceURL string // e.g. http://nginx-svc.default.svc.cluster.local:80
	CreatedAt  time.Time
}

// ClusterTunnel tracks the per-cluster cloudflared deployment and all
// hostname exposures it handles.
type ClusterTunnel struct {
	ID                     string
	ClusterID              string
	CloudflareConnectionID string
	TunnelID               string // Cloudflare tunnel UUID
	// TokenEnc is the connector token stored encrypted via SecretCipher.
	TokenEnc  string
	Namespace string // k8s namespace where cloudflared runs
	Exposures []TunnelExposure
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ClusterTunnelRepository persists ClusterTunnel aggregates.
type ClusterTunnelRepository interface {
	Save(ctx context.Context, t *ClusterTunnel) (*ClusterTunnel, error)
	Update(ctx context.Context, t *ClusterTunnel) (*ClusterTunnel, error)
	GetByClusterID(ctx context.Context, clusterID string) (*ClusterTunnel, error)
	GetByID(ctx context.Context, id string) (*ClusterTunnel, error)
	Delete(ctx context.Context, id string) error
}
