package main

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"
)

func discardLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

// TestServeStaysUpUntilTheContextIsDone pins that the shutdown budget bounds the
// cleanup, never the serving lifetime: a short budget must not stop a healthy
// server on its own.
func TestServeStaysUpUntilTheContextIsDone(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const budget = 100 * time.Millisecond
	released := make(chan struct{})
	done := make(chan error, 1)

	go func() {
		done <- serve(ctx, discardLogger(), "127.0.0.1:0", budget, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
			func() { close(released) })
	}()

	select {
	case err := <-done:
		t.Fatalf("serve returned before the context was done: %v", err)
	case <-time.After(5 * budget):
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serve returned %v after a graceful shutdown, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not return after the context was done")
	}

	select {
	case <-released:
	default:
		t.Error("serve returned without releasing the resource")
	}
}

// TestServeReportsListenFailure pins that an address the server cannot bind to
// is reported instead of being swallowed.
func TestServeReportsListenFailure(t *testing.T) {
	t.Parallel()

	err := serve(context.Background(), discardLogger(), "256.256.256.256:70000", time.Second,
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), func() {})

	if err == nil {
		t.Fatal("serve error = nil for an unbindable address, want a listen error")
	}
}
