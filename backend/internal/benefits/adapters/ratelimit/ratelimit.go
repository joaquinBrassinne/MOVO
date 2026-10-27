// Package ratelimit throttles coupon requests in memory. It implements
// benefits.GenerationLimiter; the state is per process, which is enough for the MVP,
// and the port lets Redis replace it when the API runs several instances.
package ratelimit

import (
	"sync"
	"time"
)

// Config sets the budget of every key.
type Config struct {
	// MaxAttempts is how many requests a key may make within Window.
	MaxAttempts int
	// Window is how long a key's requests count, measured from its first one.
	Window time.Duration
	// MaxKeys bounds memory: once that many keys are tracked and none has elapsed, a
	// new key is not recorded and is allowed. A flood of fresh keys cannot exhaust the
	// process, and the monthly quota of each benefit still caps the damage.
	MaxKeys int
}

type entry struct {
	attempts int
	expires  time.Time
}

// Limiter is a fixed-window counter, safe for concurrent use.
type Limiter struct {
	cfg Config
	now func() time.Time

	mu      sync.Mutex
	entries map[string]entry
}

// New builds a Limiter. A nil now uses time.Now.
func New(cfg Config, now func() time.Time) *Limiter {
	if now == nil {
		now = time.Now
	}
	return &Limiter{cfg: cfg, now: now, entries: make(map[string]entry)}
}

// Allow counts one attempt of key. Once the budget of the window is spent it refuses
// without counting further, and says how long until the window ends.
func (l *Limiter) Allow(key string) (time.Duration, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	current, ok := l.entries[key]
	if ok && !now.Before(current.expires) {
		delete(l.entries, key)
		ok = false
	}
	if !ok {
		if len(l.entries) >= l.cfg.MaxKeys {
			l.sweep(now)
		}
		if len(l.entries) >= l.cfg.MaxKeys {
			return 0, true
		}
		current = entry{expires: now.Add(l.cfg.Window)}
	}

	if current.attempts >= l.cfg.MaxAttempts {
		return current.expires.Sub(now), false
	}
	current.attempts++
	l.entries[key] = current
	return 0, true
}

// sweep drops every elapsed entry. The caller holds the lock.
func (l *Limiter) sweep(now time.Time) {
	for key, current := range l.entries {
		if !now.Before(current.expires) {
			delete(l.entries, key)
		}
	}
}
