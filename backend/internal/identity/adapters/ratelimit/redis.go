package ratelimit

import (
	"context"
	"log/slog"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/platform/cache"
)

// RedisLimiter is the Redis twin of Limiter: the same budget, counted once for every API
// instance. It implements identity.LoginLimiter and identity.RegistrationLimiter. A Redis error
// fails open: the request is allowed and a warning is logged, so Redis never blocks a login or a
// sign-up.
type RedisLimiter struct {
	window    *cache.FixedWindow
	namespace string
	log       *slog.Logger
}

// NewRedis builds a limiter whose counters live under namespace. Each limiter of the process
// (login, sign-up) needs its own namespace so their budgets stay apart. Config.MaxKeys does not
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

// Allow reports whether key may attempt a login and, when it may not, how long until it can.
func (l *RedisLimiter) Allow(key string) (time.Duration, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), cache.LimiterTimeout)
	defer cancel()

	retry, ok, err := l.window.Allow(ctx, key)
	return l.open(retry, ok, err)
}

// Attempt counts one try of key, whatever its outcome, and reports whether it is still within
// the budget.
func (l *RedisLimiter) Attempt(key string) (time.Duration, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), cache.LimiterTimeout)
	defer cancel()

	retry, ok, err := l.window.Attempt(ctx, key)
	return l.open(retry, ok, err)
}

// Fail records a failed attempt for key.
func (l *RedisLimiter) Fail(key string) {
	ctx, cancel := context.WithTimeout(context.Background(), cache.LimiterTimeout)
	defer cancel()

	if err := l.window.Fail(ctx, key); err != nil {
		l.warn(err)
	}
}

// Reset forgets the failures of key.
func (l *RedisLimiter) Reset(key string) {
	ctx, cancel := context.WithTimeout(context.Background(), cache.LimiterTimeout)
	defer cancel()

	if err := l.window.Reset(ctx, key); err != nil {
		l.warn(err)
	}
}

// open turns a Redis error into an allowed request. The key is never logged.
func (l *RedisLimiter) open(retry time.Duration, ok bool, err error) (time.Duration, bool) {
	if err != nil {
		l.warn(err)
		return 0, true
	}
	return retry, ok
}

func (l *RedisLimiter) warn(err error) {
	l.log.Warn("rate limit unavailable, failing open", "limiter", l.namespace, "err", err)
}
