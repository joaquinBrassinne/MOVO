// Package cache is the Redis-backed key-value store behind the cache decorators (SDD 7.1).
package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis is a byte store over a Redis server.
type Redis struct {
	client *redis.Client
}

// NewRedis connects to the server at url (redis://...) and checks it answers. Short timeouts
// keep a slow or dead Redis from slowing the searches that fall back to the database.
func NewRedis(ctx context.Context, url string) (*Redis, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("cache: parse REDIS_URL: %w", err)
	}
	opts.DialTimeout = 500 * time.Millisecond
	opts.ReadTimeout = 200 * time.Millisecond
	opts.WriteTimeout = 200 * time.Millisecond

	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("cache: ping redis: %w", err)
	}
	return &Redis{client: client}, nil
}

// Get returns the value at key; found is false when there is none.
func (r *Redis) Get(ctx context.Context, key string) ([]byte, bool, error) {
	value, err := r.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("cache: get %q: %w", key, err)
	}
	return value, true, nil
}

// Set stores value at key for ttl.
func (r *Redis) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := r.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("cache: set %q: %w", key, err)
	}
	return nil
}

// Incr adds one to the counter at key.
func (r *Redis) Incr(ctx context.Context, key string) error {
	if err := r.client.Incr(ctx, key).Err(); err != nil {
		return fmt.Errorf("cache: incr %q: %w", key, err)
	}
	return nil
}

// Close releases the connections.
func (r *Redis) Close() error { return r.client.Close() }
