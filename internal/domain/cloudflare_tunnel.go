package domain

import "context"

// CloudflareTunnel holds the result of a newly created Cloudflare tunnel.
type CloudflareTunnel struct {
	ID    string // UUID assigned by Cloudflare
	Name  string
	Token string // connector token — stored encrypted in ClusterTunnel
}

// CloudflareClient is the interface for Cloudflare tunnel + DNS operations.
type CloudflareClient interface {
	// CreateTunnel provisions a new Named Tunnel and returns its credentials.
	CreateTunnel(ctx context.Context, name string) (*CloudflareTunnel, error)
	// DeleteTunnel deletes the tunnel and cleans up its connectors.
	DeleteTunnel(ctx context.Context, tunnelID string) error
	// CreateCNAMERecord creates a CNAME record pointing hostname → <tunnelID>.cfargotunnel.com.
	CreateCNAMERecord(ctx context.Context, hostname, tunnelID string) error
	// DeleteDNSRecord removes the DNS record for the given hostname.
	DeleteDNSRecord(ctx context.Context, hostname string) error
	// VerifyAccess validates that the API token can access the configured account and zone.
	VerifyAccess(ctx context.Context) error
}

type CloudflareClientFactory interface {
	New(apiToken, accountID, zoneID string) CloudflareClient
}
