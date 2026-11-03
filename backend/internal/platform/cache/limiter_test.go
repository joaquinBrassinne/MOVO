package cache_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"github.com/joaquinBrassinne/MOVO/internal/platform/cache"
)

const window = 15 * time.Minute

func newCounter(t *testing.T, srv *miniredis.Miniredis, maxAttempts int) *cache.FixedWindow {
	t.Helper()
	store, err := cache.NewRedis(context.Background(), "redis://"+srv.Addr())
	if err != nil {
		t.Fatalf("NewRedis: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return cache.NewFixedWindow(store, "ratelimit:test", maxAttempts, window)
}

func TestFixedWindowAllowsUpToMaxThenBlocksWithRetryAfter(t *testing.T) {
	t.Parallel()
	srv := miniredis.RunT(t)
	ctx := context.Background()
	f := newCounter(t, srv, 3)

	for i := range 3 {
		if _, ok, err := f.Attempt(ctx, "k"); err != nil || !ok {
			t.Fatalf("attempt %d = ok %v, err %v; want allowed", i+1, ok, err)
		}
	}
	if retry, ok, err := f.Attempt(ctx, "k"); err != nil || ok || retry != window {
		t.Fatalf("4th attempt = (%v, %v, %v); want (%v, false, nil)", retry, ok, err, window)
	}

	srv.FastForward(10 * time.Minute)
	if retry, ok, _ := f.Allow(ctx, "k"); ok || retry != 5*time.Minute {
		t.Errorf("Allow after 10m = (%v, %v); want (5m, false)", retry, ok)
	}
}

func TestFixedWindowRefusedAttemptsDoNotCount(t *testing.T) {
	t.Parallel()
	srv := miniredis.RunT(t)
	ctx := context.Background()
	f := newCounter(t, srv, 2)

	for range 2 {
		_, _, _ = f.Attempt(ctx, "k")
	}
	for range 5 {
		_, _, _ = f.Attempt(ctx, "k")
	}
	keys := srv.Keys()
	if len(keys) != 1 {
		t.Fatalf("keys = %v; want one counter", keys)
	}
	if got, _ := srv.Get(keys[0]); got != "2" {
		t.Errorf("counter = %q, want 2: refused attempts must not count", got)
	}
	srv.FastForward(window)
	if _, ok, _ := f.Attempt(ctx, "k"); !ok {
		t.Error("still refused once the window elapsed")
	}
}

func TestFixedWindowKeysAreIndependent(t *testing.T) {
	t.Parallel()
	srv := miniredis.RunT(t)
	ctx := context.Background()
	f := newCounter(t, srv, 1)

	if _, ok, _ := f.Attempt(ctx, "a"); !ok {
		t.Fatal("first key refused")
	}
	if _, ok, _ := f.Attempt(ctx, "a"); ok {
		t.Error("second attempt of the same key allowed")
	}
	if _, ok, _ := f.Attempt(ctx, "b"); !ok {
		t.Error("another key must have its own budget")
	}
}

func TestFixedWindowFailAndResetLikeALogin(t *testing.T) {
	t.Parallel()
	srv := miniredis.RunT(t)
	ctx := context.Background()
	f := newCounter(t, srv, 2)

	if _, ok, _ := f.Allow(ctx, "k"); !ok {
		t.Fatal("untouched key blocked")
	}
	_ = f.Fail(ctx, "k")
	if _, ok, _ := f.Allow(ctx, "k"); !ok {
		t.Fatal("blocked after 1 failure of 2")
	}
	_ = f.Fail(ctx, "k")
	if retry, ok, _ := f.Allow(ctx, "k"); ok || retry != window {
		t.Errorf("Allow after 2 failures = (%v, %v); want (%v, false)", retry, ok, window)
	}

	if err := f.Reset(ctx, "k"); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if _, ok, _ := f.Allow(ctx, "k"); !ok {
		t.Error("still blocked after Reset")
	}
}

func TestFixedWindowWindowIsAnchoredAtTheFirstCount(t *testing.T) {
	t.Parallel()
	srv := miniredis.RunT(t)
	ctx := context.Background()
	f := newCounter(t, srv, 3)

	_ = f.Fail(ctx, "k")
	srv.FastForward(14 * time.Minute)
	_ = f.Fail(ctx, "k")
	_ = f.Fail(ctx, "k")

	if retry, ok, _ := f.Allow(ctx, "k"); ok || retry != time.Minute {
		t.Errorf("Allow = (%v, %v); want (1m, false): later failures must not extend the block", retry, ok)
	}
}

func TestFixedWindowKeysNeverLackAnExpiry(t *testing.T) {
	t.Parallel()
	srv := miniredis.RunT(t)
	ctx := context.Background()
	f := newCounter(t, srv, 5)

	_ = f.Fail(ctx, "a")
	_, _, _ = f.Attempt(ctx, "b")
	keys := srv.Keys()
	if len(keys) != 2 {
		t.Fatalf("keys = %v; want 2 counters", keys)
	}
	for _, k := range keys {
		if ttl := srv.TTL(k); ttl != window {
			t.Errorf("key %q ttl = %v; want %v", k, ttl, window)
		}
	}
}

func TestFixedWindowHidesTheRawKey(t *testing.T) {
	t.Parallel()
	srv := miniredis.RunT(t)
	f := newCounter(t, srv, 5)

	_ = f.Fail(context.Background(), "someone@example.com")
	for _, k := range srv.Keys() {
		if strings.Contains(k, "someone@example.com") {
			t.Fatalf("redis key %q contains the raw identifier", k)
		}
	}
}

func TestFixedWindowAttemptNeverPassesTheBudgetWhenManyArriveTogether(t *testing.T) {
	t.Parallel()
	srv := miniredis.RunT(t)
	f := newCounter(t, srv, 5)

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		allowed int
	)
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, ok, _ := f.Attempt(context.Background(), "k"); ok {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if allowed != 5 {
		t.Errorf("allowed = %d, want exactly 5", allowed)
	}
}

func TestFixedWindowReportsErrorsWhenRedisIsDown(t *testing.T) {
	srv := miniredis.RunT(t)
	ctx := context.Background()
	f := newCounter(t, srv, 1)
	srv.Close()

	if _, _, err := f.Attempt(ctx, "k"); err == nil {
		t.Error("Attempt succeeded with redis down")
	}
	if _, _, err := f.Allow(ctx, "k"); err == nil {
		t.Error("Allow succeeded with redis down")
	}
	if err := f.Fail(ctx, "k"); err == nil {
		t.Error("Fail succeeded with redis down")
	}
	if err := f.Reset(ctx, "k"); err == nil {
		t.Error("Reset succeeded with redis down")
	}
}
