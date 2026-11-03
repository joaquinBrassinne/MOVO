package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// LimiterTimeout bounds one limiter round trip. A slow Redis then costs a request at most this
// much before the caller fails open.
const LimiterTimeout = 300 * time.Millisecond

// FixedWindow counts the attempts of each key in a fixed window kept in Redis, so every API
// instance counts into the same budget (SDD 7.3). The window of a key starts with its first
// counted attempt and lasts window; the key expires with it, so memory stays bounded without a
// cap on the number of keys.
type FixedWindow struct {
	client *redis.Client
	prefix string
	max    int
	window time.Duration
}

// NewFixedWindow builds a counter whose keys live under prefix (for example "ratelimit:login").
// Keys are hashed before they reach Redis, so no raw email or address is stored there.
func NewFixedWindow(r *Redis, prefix string, maxAttempts int, window time.Duration) *FixedWindow {
	return &FixedWindow{client: r.client, prefix: prefix, max: maxAttempts, window: window}
}

// The scripts run atomically on the server: a count and its expiry are never apart, and two
// instances racing on the same key cannot pass the budget. A script returns {allowed, pttl_ms}.
var (
	// checkScript reads the counter without counting.
	checkScript = redis.NewScript(`
local n = tonumber(redis.call('GET', KEYS[1]) or '0')
if n >= tonumber(ARGV[1]) then
  return {0, redis.call('PTTL', KEYS[1])}
end
return {1, 0}
`)
	// attemptScript counts one try, unless the budget is already spent, in which case it refuses
	// without counting further, as the in-memory limiter does.
	attemptScript = redis.NewScript(`
local n = tonumber(redis.call('GET', KEYS[1]) or '0')
if n >= tonumber(ARGV[1]) then
  return {0, redis.call('PTTL', KEYS[1])}
end
redis.call('INCR', KEYS[1])
if redis.call('PTTL', KEYS[1]) == -1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[2])
end
return {1, redis.call('PTTL', KEYS[1])}
`)
	// failScript counts one failure, the way a failed login does; the window starts with the first.
	failScript = redis.NewScript(`
redis.call('INCR', KEYS[1])
if redis.call('PTTL', KEYS[1]) == -1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return 1
`)
)

// Allow reports whether key may try again and, when it may not, how long until its window ends.
// It does not count an attempt; Fail does.
func (f *FixedWindow) Allow(ctx context.Context, key string) (time.Duration, bool, error) {
	return f.decide(ctx, checkScript, key)
}

// Attempt counts one try of key and reports whether it is within the budget. Once the budget is
// spent it refuses without counting further, and says how long until the window ends.
func (f *FixedWindow) Attempt(ctx context.Context, key string) (time.Duration, bool, error) {
	return f.decide(ctx, attemptScript, key)
}

// Fail counts one failed attempt of key.
func (f *FixedWindow) Fail(ctx context.Context, key string) error {
	if err := failScript.Run(ctx, f.client, []string{f.keyOf(key)}, f.window.Milliseconds()).Err(); err != nil {
		return fmt.Errorf("cache: rate limit %q: fail: %w", f.prefix, err)
	}
	return nil
}

// Reset forgets the count of key.
func (f *FixedWindow) Reset(ctx context.Context, key string) error {
	if err := f.client.Del(ctx, f.keyOf(key)).Err(); err != nil {
		return fmt.Errorf("cache: rate limit %q: reset: %w", f.prefix, err)
	}
	return nil
}

func (f *FixedWindow) decide(ctx context.Context, script *redis.Script, key string) (time.Duration, bool, error) {
	reply, err := script.Run(ctx, f.client, []string{f.keyOf(key)}, f.max, f.window.Milliseconds()).Int64Slice()
	if err != nil {
		return 0, false, fmt.Errorf("cache: rate limit %q: %w", f.prefix, err)
	}
	if len(reply) != 2 {
		return 0, false, fmt.Errorf("cache: rate limit %q: unexpected reply of %d values", f.prefix, len(reply))
	}
	if reply[0] == 1 {
		return 0, true, nil
	}
	if reply[1] < 0 {
		// The scripts always set a TTL, so this is only a defensive answer: the full window.
		return f.window, false, nil
	}
	return time.Duration(reply[1]) * time.Millisecond, false, nil
}

func (f *FixedWindow) keyOf(key string) string {
	sum := sha256.Sum256([]byte(key))
	return f.prefix + ":" + hex.EncodeToString(sum[:])
}
