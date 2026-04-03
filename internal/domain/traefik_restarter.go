package domain

import "context"

// TraefikRestarter can restart the Traefik container so that static config
// changes (e.g. new ACME email) take effect.
type TraefikRestarter interface {
	RestartTraefik(ctx context.Context) error
}
