package ratelimit_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"

	"github.com/joaquinBrassinne/MOVO/internal/identity/adapters/ratelimit"
	"github.com/joaquinBrassinne/MOVO/internal/platform/cache"
)

func newRedisStore(t *testing.T, srv *miniredis.Miniredis) *cache.Redis {
	t.Helper()
	store, err := cache.NewRedis(context.Background(), "redis://"+srv.Addr())
	if err != nil {
		t.Fatalf("NewRedis: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func redisLimiterConfig() ratelimit.Config {
	return ratelimit.Config{MaxAttempts: 5, Window: window, MaxKeys: 100}
}

func TestRedisLimiterBlocksAfterTheFailureBudgetWithRetryAfter(t *testing.T) {
	t.Parallel()
	srv := miniredis.RunT(t)
	l := ratelimit.NewRedis(newRedisStore(t, srv), "login", redisLimiterConfig(), nil)

	for range 4 {
		l.Fail("email:a@x.com")
	}
	if _, ok := l.Allow("email:a@x.com"); !ok {
		t.Fatal("blocked after 4 failures, want the 5th attempt allowed")
	}
	l.Fail("email:a@x.com")
	if retry, ok := l.Allow("email:a@x.com"); ok || retry != window {
		t.Errorf("Allow after 5 failures = (%v, %v), want (%v, false)", retry, ok, window)
	}

	l.Reset("email:a@x.com")
	if _, ok := l.Allow("email:a@x.com"); !ok {
		t.Error("still blocked after Reset")
	}
}

func TestRedisLimiterSharesTheBudgetAcrossInstances(t *testing.T) {
	t.Parallel()
	srv := miniredis.RunT(t)
	instanceA := ratelimit.NewRedis(newRedisStore(t, srv), "login", redisLimiterConfig(), nil)
	instanceB := ratelimit.NewRedis(newRedisStore(t, srv), "login", redisLimiterConfig(), nil)

	for range 5 {
		instanceA.Fail("ip:10.0.0.1")
	}
	if retry, ok := instanceB.Allow("ip:10.0.0.1"); ok || retry != window {
		t.Errorf("instance B Allow = (%v, %v), want (%v, false): failures on A must count on B", retry, ok, window)
	}
}

func TestRedisLimiterNamespacesKeepBudgetsApart(t *testing.T) {
	t.Parallel()
	srv := miniredis.RunT(t)
	store := newRedisStore(t, srv)
	login := ratelimit.NewRedis(store, "login", redisLimiterConfig(), nil)
	signUps := ratelimit.NewRedis(store, "signup", redisLimiterConfig(), nil)

	for range 5 {
		signUps.Attempt("ip:10.0.0.1")
	}
	if _, ok := signUps.Attempt("ip:10.0.0.1"); ok {
		t.Fatal("sign-up budget not spent")
	}
	if _, ok := login.Allow("ip:10.0.0.1"); !ok {
		t.Error("a burst of sign-ups blocked the login of the same address")
	}
}

func TestRedisLimiterFailsOpenWhenRedisIsDown(t *testing.T) {
	srv := miniredis.RunT(t)
	store := newRedisStore(t, srv)
	var logs bytes.Buffer
	l := ratelimit.NewRedis(store, "login", redisLimiterConfig(), slog.New(slog.NewTextHandler(&logs, nil)))
	srv.Close()

	if _, ok := l.Allow("email:a@x.com"); !ok {
		t.Error("Allow refused with redis down, want it to fail open")
	}
	if _, ok := l.Attempt("ip:10.0.0.1"); !ok {
		t.Error("Attempt refused with redis down, want it to fail open")
	}
	l.Fail("email:a@x.com")
	l.Reset("email:a@x.com")

	out := logs.String()
	if !strings.Contains(out, "failing open") || !strings.Contains(out, "limiter=login") {
		t.Errorf("log = %q, want a warning naming the limiter", out)
	}
	if strings.Contains(out, "a@x.com") || strings.Contains(out, "10.0.0.1") {
		t.Errorf("log leaks a key: %q", out)
	}
}
