package services

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"golang.org/x/sync/singleflight"
)

var ErrCacheMiss = errors.New("cache miss")

var cacheSingleflightGroup singleflight.Group

type Cache interface {
	Get(ctx context.Context, key string, dest any) error
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

func GetOrCreate[T any](
	ctx context.Context,
	c Cache,
	key string,
	ttl time.Duration,
	fetch func(ctx context.Context) (T, error),
) (T, error) {
	var zero T
	if c == nil {
		return fetch(ctx)
	}

	var dest T
	if err := c.Get(ctx, key, &dest); err == nil {
		slog.InfoContext(ctx, "cache hit", "key", key)
		return dest, nil
	} else if !errors.Is(err, ErrCacheMiss) {
		return zero, err
	}

	result, err, _ := cacheSingleflightGroup.Do(key, func() (any, error) {
		var cached T
		if err := c.Get(ctx, key, &cached); err == nil {
			slog.InfoContext(ctx, "cache hit", "key", key)
			return cached, nil
		} else if !errors.Is(err, ErrCacheMiss) {
			return zero, err
		}

		value, err := fetch(ctx)
		if err != nil {
			return zero, err
		}

		if err := c.Set(ctx, key, value, ttl); err != nil {
			slog.WarnContext(ctx, "cache set failed", "key", key, "err", err)
		}

		return value, nil
	})
	if err != nil {
		return zero, err
	}
	value, ok := result.(T)
	if !ok {
		return zero, errors.New("cache singleflight type assertion failed")
	}

	return value, nil
}
