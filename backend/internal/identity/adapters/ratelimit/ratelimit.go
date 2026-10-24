// Package ratelimit throttles failed logins in memory. It implements
// identity.LoginLimiter; the state is per process, which is enough for the
// MVP, and the port lets Redis replace it when the API runs several instances.
package ratelimit

import (
	"sync"
	"time"
)

// Config sets the failure budget of every key.
type Config struct {
	// MaxAttempts is how many failures a key may accumulate within Window
	// before it is blocked.
	MaxAttempts int
	// Window is how long a key's failures count, measured from its first one.
	Window time.Duration
	// MaxKeys bounds memory: once that many keys are tracked, new keys are not
	// recorded. A flood of fresh keys cannot exhaust the process, and it gains
	// nothing, because a key with no failures is never blocked.
	MaxKeys int
}

type entry struct {
	failures int
	expires  time.Time
}

// Limiter is safe for concurrent use.
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

// Allow reports whether key may attempt a login and, when it may not, how long
// until it can.
func (l *Limiter) Allow(key string) (time.Duration, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	current, ok := l.live(key)
	if !ok || current.failures < l.cfg.MaxAttempts {
		return 0, true
	}

	return current.expires.Sub(l.now()), false
}

// Fail records a failed attempt for key.
func (l *Limiter) Fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	current, ok := l.live(key)
	if !ok {
		if len(l.entries) >= l.cfg.MaxKeys {
			l.sweep()
		}
		if len(l.entries) >= l.cfg.MaxKeys {
			return
		}
		current = entry{expires: l.now().Add(l.cfg.Window)}
	}

	current.failures++
	l.entries[key] = current
}

// Reset forgets the failures of key.
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.entries, key)
}

// live returns the entry of key unless its window has already elapsed. The
// caller holds the lock.
func (l *Limiter) live(key string) (entry, bool) {
	current, ok := l.entries[key]
	if !ok {
		return entry{}, false
	}
	if !l.now().Before(current.expires) {
		delete(l.entries, key)
		return entry{}, false
	}

	return current, true
}

// sweep drops every elapsed entry. The caller holds the lock.
func (l *Limiter) sweep() {
	now := l.now()
	for key, current := range l.entries {
		if !now.Before(current.expires) {
			delete(l.entries, key)
		}
	}
}
