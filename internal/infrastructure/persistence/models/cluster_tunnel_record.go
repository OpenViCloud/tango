package models

import "time"

// ClusterTunnelRecord stores one cloudflared deployment per cluster.
type ClusterTunnelRecord struct {
	ID                     string    `gorm:"primaryKey;type:text"`
	ClusterID              string    `gorm:"column:cluster_id;type:text;not null;uniqueIndex"`
	CloudflareConnectionID string    `gorm:"column:cloudflare_connection_id;type:text;not null;index"`
	TunnelID               string    `gorm:"column:tunnel_id;type:text;not null"`
	TokenEnc               string    `gorm:"column:token_enc;type:text;not null"`
	Namespace              string    `gorm:"column:namespace;type:text;not null"`
	CreatedAt              time.Time `gorm:"column:created_at;not null"`
	UpdatedAt              time.Time `gorm:"column:updated_at;not null"`

	Exposures []TunnelExposureRecord `gorm:"foreignKey:ClusterTunnelID;constraint:OnDelete:CASCADE"`
}

func (ClusterTunnelRecord) TableName() string { return "cluster_tunnels" }

// TunnelExposureRecord stores one hostname → service mapping inside a tunnel.
type TunnelExposureRecord struct {
	ID              string    `gorm:"primaryKey;type:text"`
	ClusterTunnelID string    `gorm:"column:cluster_tunnel_id;type:text;not null;index"`
	Hostname        string    `gorm:"column:hostname;type:text;not null"`
	ServiceURL      string    `gorm:"column:service_url;type:text;not null"`
	CreatedAt       time.Time `gorm:"column:created_at;not null"`
}

func (TunnelExposureRecord) TableName() string { return "tunnel_exposures" }
