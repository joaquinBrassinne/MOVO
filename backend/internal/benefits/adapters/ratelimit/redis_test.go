package ratelimit_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/adapters/ratelimit"
	"github.com/joaquinBrassinne/MOVO/internal/platform/cache"
)

func newCouponStore(t *testing.T, srv *miniredis.Miniredis) *cache.Redis {
	t.Helper()
	store, err := cache.NewRedis(context.Background(), "redis://"+srv.Addr())
	if err != nil {
		t.Fatalf("NewRedis: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestRedisAllowsTheBudgetThenRefusesWithRetryAfter(t *testing.T) {
	t.Parallel()
	srv := miniredis.RunT(t)
	l := ratelimit.NewRedis(newCouponStore(t, srv), "coupon", ratelimit.Config{MaxAttempts: 3, Window: window}, nil)

	for i := range 3 {
		if _, ok := l.Allow("ip:10.0.0.1"); !ok {
			t.Fatalf("attempt %d refused, want the budget of 3 allowed", i+1)
		}
	}
	if retry, ok := l.Allow("ip:10.0.0.1"); ok || retry != window {
		t.Errorf("4th attempt = (%v, %v), want (%v, false)", retry, ok, window)
	}
	if _, ok := l.Allow("ip:10.0.0.2"); !ok {
		t.Error("another origin must have its own budget")
	}
}

func TestRedisBudgetIsSharedAcrossInstances(t *testing.T) {
	t.Parallel()
	srv := miniredis.RunT(t)
	cfg := ratelimit.Config{MaxAttempts: 2, Window: window}
	instanceA := ratelimit.NewRedis(newCouponStore(t, srv), "coupon", cfg, nil)
	instanceB := ratelimit.NewRedis(newCouponStore(t, srv), "coupon", cfg, nil)

	instanceA.Allow("ip:10.0.0.1")
	instanceB.Allow("ip:10.0.0.1")
	if _, ok := instanceA.Allow("ip:10.0.0.1"); ok {
		t.Error("the third request across two instances was allowed, want the shared budget of 2 spent")
	}
}

func TestRedisFailsOpenWhenRedisIsDown(t *testing.T) {
	srv := miniredis.RunT(t)
	var logs bytes.Buffer
	l := ratelimit.NewRedis(newCouponStore(t, srv), "coupon", ratelimit.Config{MaxAttempts: 1, Window: window},
		slog.New(slog.NewTextHandler(&logs, nil)))
	srv.Close()

	for range 3 {
		if _, ok := l.Allow("ip:10.0.0.1"); !ok {
			t.Fatal("a coupon was refused with redis down, want it to fail open")
		}
	}
	if out := logs.String(); !strings.Contains(out, "failing open") || strings.Contains(out, "10.0.0.1") {
		t.Errorf("log = %q, want a warning that names no key", out)
	}
}
