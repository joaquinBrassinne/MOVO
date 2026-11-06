package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

// A task that hangs (a stuck storage call, a locked table) must not hold its advisory
// lock and block every later round: it is cut off at the limit.
func TestBoundedCutsATaskOffAtTheLimit(t *testing.T) {
	t.Parallel()

	stuck := func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}

	err := bounded(20*time.Millisecond, stuck)(t.Context())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context.DeadlineExceeded", err)
	}
}

func TestBoundedLeavesAFastTaskAlone(t *testing.T) {
	t.Parallel()

	want := errors.New("boom")
	if err := bounded(time.Minute, func(context.Context) error { return nil })(t.Context()); err != nil {
		t.Errorf("error = %v, want nil", err)
	}
	if err := bounded(time.Minute, func(context.Context) error { return want })(t.Context()); !errors.Is(err, want) {
		t.Errorf("error = %v, want %v", err, want)
	}
}
