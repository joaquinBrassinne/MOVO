package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"github.com/joaquinBrassinne/MOVO/internal/platform/cache"
)

func TestRedisStoresAndExpiresValues(t *testing.T) {
	srv := miniredis.RunT(t)
	ctx := context.Background()
	store, err := cache.NewRedis(ctx, "redis://"+srv.Addr())
	if err != nil {
		t.Fatalf("NewRedis: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	if _, found, err := store.Get(ctx, "k"); err != nil || found {
		t.Fatalf("Get on empty = found %v, err %v; want a clean miss", found, err)
	}
	if err := store.Set(ctx, "k", []byte("v"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, found, err := store.Get(ctx, "k")
	if err != nil || !found || string(got) != "v" {
		t.Fatalf("Get = %q, %v, %v; want v, true, nil", got, found, err)
	}
	srv.FastForward(2 * time.Minute)
	if _, found, _ := store.Get(ctx, "k"); found {
		t.Fatal("entry still there after its TTL")
	}
}

func TestRedisIncrCountsUp(t *testing.T) {
	srv := miniredis.RunT(t)
	ctx := context.Background()
	store, err := cache.NewRedis(ctx, "redis://"+srv.Addr())
	if err != nil {
		t.Fatalf("NewRedis: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	for range 2 {
		if err := store.Incr(ctx, "n"); err != nil {
			t.Fatalf("Incr: %v", err)
		}
	}
	if got, _ := srv.Get("n"); got != "2" {
		t.Errorf("counter = %q, want 2", got)
	}
}

func TestRedisReportsAnUnreachableServer(t *testing.T) {
	if _, err := cache.NewRedis(context.Background(), "redis://127.0.0.1:1"); err == nil {
		t.Fatal("NewRedis against a closed port succeeded")
	}
	if _, err := cache.NewRedis(context.Background(), "not a url"); err == nil {
		t.Fatal("NewRedis accepted a malformed URL")
	}
}

func TestRedisErrorsAfterTheServerGoesAway(t *testing.T) {
	srv := miniredis.RunT(t)
	ctx := context.Background()
	store, err := cache.NewRedis(ctx, "redis://"+srv.Addr())
	if err != nil {
		t.Fatalf("NewRedis: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	srv.Close()

	if _, _, err := store.Get(ctx, "k"); err == nil {
		t.Fatal("Get succeeded with the server down")
	}
	if err := store.Set(ctx, "k", []byte("v"), time.Minute); err == nil {
		t.Fatal("Set succeeded with the server down")
	}
}
