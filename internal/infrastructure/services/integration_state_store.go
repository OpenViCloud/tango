package services

import (
	"context"
	"fmt"
	"time"

	appservices "tango/internal/application/services"
)

type cacheIntegrationStateStore struct {
	cache appservices.Cache
}

func NewIntegrationStateStore(cache appservices.Cache) appservices.IntegrationStateStore {
	return &cacheIntegrationStateStore{cache: cache}
}

func (s *cacheIntegrationStateStore) Save(ctx context.Context, state string, value appservices.SourceIntegrationState, ttl time.Duration) error {
	if s.cache == nil {
		return fmt.Errorf("integration state store cache is not configured")
	}
	return s.cache.Set(ctx, integrationStateKey(state), value, ttl)
}

func (s *cacheIntegrationStateStore) Consume(ctx context.Context, state string) (*appservices.SourceIntegrationState, error) {
	if s.cache == nil {
		return nil, fmt.Errorf("integration state store cache is not configured")
	}
	var value appservices.SourceIntegrationState
	if err := s.cache.Get(ctx, integrationStateKey(state), &value); err != nil {
		return nil, err
	}
	if err := s.cache.Delete(ctx, integrationStateKey(state)); err != nil {
		return nil, err
	}
	return &value, nil
}

func integrationStateKey(state string) string {
	return "integration_state:" + state
}
