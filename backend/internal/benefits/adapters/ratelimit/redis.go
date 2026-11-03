package ratelimit

import (
	"context"
	"log/slog"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/platform/cache"
)

// RedisLimiter is the Redis twin of Limiter: the same coupon budget, counted once for every API
// instance. It implements benefits.GenerationLimiter. A Redis error fails open: the request is
// allowed and a warning is logged, so Redis never blocks a coupon.
type RedisLimiter struct {
	window    *cache.FixedWindow
	namespace string
	log       *slog.Logger
}

// NewRedis builds a coupon limiter whose counters live under namespace. Config.MaxKeys does not
// apply here: Redis expires every counter with its window, which bounds memory on its own.
func NewRedis(store *cache.Redis, namespace string, cfg Config, log *slog.Logger) *RedisLimiter {
	if log == nil {
		log = slog.Default()
	}
	return &RedisLimiter{
		window:    cache.NewFixedWindow(store, "ratelimit:"+namespace, cfg.MaxAttempts, cfg.Window),
		namespace: namespace,
		log:       log,
	}
}

// Allow counts one attempt of key, successful or not, and reports whether it may go on. When it
// may not, retryAfter is how long until the key can try again.
func (l *RedisLimiter) Allow(key string) (time.Duration, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), cache.LimiterTimeout)
	defer cancel()

	retry, ok, err := l.window.Attempt(ctx, key)
	if err != nil {
		l.log.Warn("rate limit unavailable, failing open", "limiter", l.namespace, "err", err)
		return 0, true
	}
	return retry, ok
}
