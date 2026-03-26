package cache

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	appservices "tango/internal/application/services"
)

type memoryItem struct {
	value     []byte
	expiresAt time.Time
}

type MemoryCache struct {
	mu         sync.RWMutex
	items      map[string]memoryItem
	defaultTTL time.Duration
}

func NewMemoryCache(defaultTTL time.Duration) *MemoryCache {
	if defaultTTL <= 0 {
		defaultTTL = time.Minute
	}
	return &MemoryCache{
		items:      make(map[string]memoryItem),
		defaultTTL: defaultTTL,
	}
}

func (c *MemoryCache) Get(_ context.Context, key string, dest any) error {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return appservices.ErrCacheMiss
	}
	if time.Now().UTC().After(item.expiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return appservices.ErrCacheMiss
	}
	if err := json.Unmarshal(item.value, dest); err != nil {
		return err
	}
	return nil
}

func (c *MemoryCache) Set(_ context.Context, key string, value any, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = c.defaultTTL
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.items[key] = memoryItem{
		value:     data,
		expiresAt: time.Now().UTC().Add(ttl),
	}
	c.mu.Unlock()
	return nil
}

func (c *MemoryCache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
	return nil
}

var _ appservices.Cache = (*MemoryCache)(nil)
