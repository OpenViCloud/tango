package cloudflare

import "tango/internal/domain"

type Factory struct{}

func NewFactory() *Factory {
	return &Factory{}
}

func (f *Factory) New(apiToken, accountID, zoneID string) domain.CloudflareClient {
	return New(apiToken, accountID, zoneID)
}

var _ domain.CloudflareClientFactory = (*Factory)(nil)
