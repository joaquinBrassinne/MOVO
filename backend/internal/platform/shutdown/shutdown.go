// Package shutdown runs ordered cleanup hooks during a graceful shutdown.
package shutdown

import (
	"context"
	"log/slog"
	"time"
)

// DefaultTimeout bounds the shutdown budget when the caller provides no usable
// deadline. It matches the config default for SHUTDOWN_TIMEOUT.
const DefaultTimeout = 10 * time.Second

// Hook is a cleanup task run during shutdown. It receives a context bounded by
// the shutdown budget and should return as soon as that context is done.
type Hook func(ctx context.Context) error

// Run blocks until ctx is done and then runs the hooks in reverse registration
// order, so the most recently registered resource is released first.
//
// The configured shutdown budget comes from ctx: callers derive it from
// Config.ShutdownTimeout (for example
// context.WithTimeout(ctx, cfg.ShutdownTimeout)), and every hook receives a
// context bounded by the time remaining on that deadline, or by DefaultTimeout
// when ctx carries no deadline.
//
// A hook that fails or panics is logged and the remaining hooks still run, so
// Run never panics and never leaves later cleanup work unexecuted.
func Run(ctx context.Context, logger *slog.Logger, hooks ...Hook) {
	if ctx == nil {
		// A nil context can never signal completion: treat it as already done
		// instead of blocking forever.
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(context.Background())
		cancel()
	}
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	budget := shutdownBudget(ctx)
	<-ctx.Done()

	// The signal context is done by now, so detach from its cancellation while
	// keeping its values and bound the hooks with a fresh deadline.
	hookCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), budget)
	defer cancel()

	for i := len(hooks) - 1; i >= 0; i-- {
		runHook(hookCtx, logger, i, hooks[i])
	}
}

// shutdownBudget returns the remaining time on ctx's deadline, or
// DefaultTimeout when ctx has no deadline left to spend.
func shutdownBudget(ctx context.Context) time.Duration {
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining > 0 {
			return remaining
		}
	}
	return DefaultTimeout
}

// runHook executes one hook, logging (instead of propagating) failures and
// panics so the loop keeps going.
func runHook(ctx context.Context, logger *slog.Logger, index int, hook Hook) {
	if hook == nil {
		return
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			logger.Error("shutdown hook panicked", "hook", index, "panic", recovered)
		}
	}()

	if err := hook(ctx); err != nil {
		logger.Error("shutdown hook failed", "hook", index, "error", err)
	}
}
