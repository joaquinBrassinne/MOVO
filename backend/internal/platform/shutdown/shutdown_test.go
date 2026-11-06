package shutdown_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/platform/logger"
	"github.com/joaquinBrassinne/MOVO/internal/platform/shutdown"
)

// doneContext returns an already-cancelled context that still carries a
// deadline of budget, mimicking a caller that derived its shutdown context
// from the configured SHUTDOWN_TIMEOUT.
func doneContext(t *testing.T, budget time.Duration) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), budget)
	t.Cleanup(cancel)
	cancel()
	return ctx
}

// recorder collects the hook labels that actually ran.
type recorder struct {
	mu     sync.Mutex
	labels []string
}

func (r *recorder) add(label string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.labels = append(r.labels, label)
}

func (r *recorder) got() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.labels...)
}

// TestRunWaitsForContextDone covers S7: no hook runs and Run does not return
// before ctx is done.
func TestRunWaitsForContextDone(t *testing.T) {
	var ran recorder
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	returned := make(chan struct{})
	go func() {
		shutdown.Run(ctx, nil, func(context.Context) error {
			ran.add("hook")
			return nil
		})
		close(returned)
	}()

	// Give Run the chance to run the hook too early.
	time.Sleep(50 * time.Millisecond)

	if labels := ran.got(); len(labels) != 0 {
		t.Fatalf("hooks ran before ctx was done: %v", labels)
	}
	select {
	case <-returned:
		t.Fatal("Run returned before ctx was done")
	default:
	}

	cancel()

	select {
	case <-returned:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after ctx was done")
	}
	if labels := ran.got(); len(labels) != 1 {
		t.Fatalf("ran hooks = %v, want 1 hook", labels)
	}
}

// TestRunRunsHooksInReverseOrder covers S7.
func TestRunRunsHooksInReverseOrder(t *testing.T) {
	var ran recorder

	shutdown.Run(doneContext(t, 5*time.Second), nil,
		func(context.Context) error { ran.add("first"); return nil },
		func(context.Context) error { ran.add("second"); return nil },
		func(context.Context) error { ran.add("third"); return nil },
	)

	want := []string{"third", "second", "first"}
	got := ran.got()
	if len(got) != len(want) {
		t.Fatalf("ran hooks = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ran hooks = %v, want %v", got, want)
		}
	}
}

// TestRunBoundsHooksByConfiguredTimeout covers S7: each hook receives a context
// bounded by the configured timeout, so a hook that waits for it is released
// instead of blocking the shutdown forever.
func TestRunBoundsHooksByConfiguredTimeout(t *testing.T) {
	const budget = 100 * time.Millisecond

	var (
		deadlineSeen bool
		hookErr      error
	)

	start := time.Now()
	shutdown.Run(doneContext(t, budget), nil, func(ctx context.Context) error {
		if _, ok := ctx.Deadline(); ok {
			deadlineSeen = true
			<-ctx.Done()
			hookErr = ctx.Err()
		}
		return nil
	})
	elapsed := time.Since(start)

	if !deadlineSeen {
		t.Fatal("hook context had no deadline, want one derived from the configured timeout")
	}
	if !errors.Is(hookErr, context.DeadlineExceeded) {
		t.Errorf("hook context error = %v, want %v", hookErr, context.DeadlineExceeded)
	}
	if elapsed > 2*time.Second {
		t.Errorf("Run took %v, want it bounded near the configured timeout %v", elapsed, budget)
	}
}

// TestRunLogsFailingHookAndKeepsGoing covers S7: a failing hook is logged and
// the remaining hooks still run.
func TestRunLogsFailingHookAndKeepsGoing(t *testing.T) {
	var (
		buf bytes.Buffer
		ran recorder
	)

	shutdown.Run(doneContext(t, 5*time.Second), logger.New("info", &buf),
		func(context.Context) error { ran.add("after-failure"); return nil },
		func(context.Context) error { return errors.New("database still busy") },
	)

	if want := []string{"after-failure"}; len(ran.got()) != 1 || ran.got()[0] != want[0] {
		t.Fatalf("ran hooks = %v, want %v (hooks after the failure must run)", ran.got(), want)
	}

	output := buf.String()
	if !strings.Contains(output, "database still busy") {
		t.Errorf("log output does not report the hook failure; output = %q", output)
	}
	if !strings.Contains(output, `"level":"ERROR"`) {
		t.Errorf("log output does not contain an ERROR record; output = %q", output)
	}
}

// TestRunRecoversHookPanics covers S7: Run never panics, even when a hook does.
func TestRunRecoversHookPanics(t *testing.T) {
	var (
		buf bytes.Buffer
		ran recorder
	)

	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Run panicked: %v", r)
			}
		}()
		shutdown.Run(doneContext(t, 5*time.Second), logger.New("info", &buf),
			func(context.Context) error { ran.add("after-panic"); return nil },
			func(context.Context) error { panic("kaboom") },
		)
	}()

	if got := ran.got(); len(got) != 1 || got[0] != "after-panic" {
		t.Fatalf("ran hooks = %v, want [after-panic] (hooks after the panic must run)", got)
	}
	if !strings.Contains(buf.String(), "kaboom") {
		t.Errorf("log output does not report the panic; output = %q", buf.String())
	}
}

// TestRunToleratesMinimalInput covers S7: no hooks, a nil logger and a nil hook
// are all safe.
func TestRunToleratesMinimalInput(t *testing.T) {
	shutdown.Run(doneContext(t, 100*time.Millisecond), nil)

	var called recorder
	shutdown.Run(doneContext(t, 100*time.Millisecond), nil, nil, func(context.Context) error {
		called.add("hook")
		return nil
	})
	if got := called.got(); len(got) != 1 {
		t.Fatalf("ran hooks = %v, want 1 hook", got)
	}
}

// TestRunToleratesNilContext covers S7: a nil context can never signal
// completion, so Run runs the hooks instead of blocking or panicking.
func TestRunToleratesNilContext(t *testing.T) {
	var ran recorder

	//nolint:staticcheck // passing a nil context is exactly what this test covers.
	shutdown.Run(nil, nil, func(context.Context) error {
		ran.add("hook")
		return nil
	})

	if got := ran.got(); len(got) != 1 {
		t.Fatalf("ran hooks = %v, want 1 hook", got)
	}
}

// TestRunHandlesNilLoggerForFailingHook covers S7: logging a failure without a
// logger must not panic.
func TestRunHandlesNilLoggerForFailingHook(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Run panicked with a nil logger: %v", r)
		}
	}()

	var emptyLogger *slog.Logger
	shutdown.Run(doneContext(t, 100*time.Millisecond), emptyLogger, func(context.Context) error {
		return errors.New("boom")
	})
}
