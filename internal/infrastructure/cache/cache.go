package cache

import (
	"fmt"
	"time"

	appservices "tango/internal/application/services"
	"tango/internal/config"
)

func New(cfg *config.Config) (appservices.Cache, error) {
	driver := "memory"
	ttl := time.Minute
	if cfg != nil {
		if cfg.CacheDriver != "" {
			driver = cfg.CacheDriver
		}
		if cfg.CacheDefaultTTL > 0 {
			ttl = cfg.CacheDefaultTTL
		}
	}

	switch driver {
	case "memory":
		return NewMemoryCache(ttl), nil
	case "redis":
		return nil, fmt.Errorf("cache driver redis is not implemented")
	default:
		return nil, fmt.Errorf("unsupported cache driver: %s", driver)
	}
}
