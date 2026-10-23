// Command api serves the MOVO HTTP API.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
	"github.com/joaquinBrassinne/MOVO/internal/platform/db"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
	"github.com/joaquinBrassinne/MOVO/internal/platform/logger"
	"github.com/joaquinBrassinne/MOVO/internal/platform/shutdown"
)

// HTTP server timeouts; the shutdown budget is separate and comes from config.
const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 60 * time.Second
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "api: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}

	log := logger.New(cfg.LogLevel, os.Stdout)

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(signalCtx, cfg.DatabaseURL)
	if err != nil {
		return err
	}

	return serve(signalCtx, log, cfg.HTTPAddr, cfg.ShutdownTimeout, httpx.NewRouter(log, pool), pool.Close)
}

// serve runs the HTTP server until ctx is done and then shuts it down within
// timeout. Only ctx ends the serving lifetime: the budget bounds the cleanup.
func serve(
	ctx context.Context,
	log *slog.Logger,
	addr string,
	timeout time.Duration,
	handler http.Handler,
	release func(),
) error {
	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Info("http server listening", "addr", addr)
		serveErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		release()
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
	}

	log.Info("shutdown signal received")

	// shutdown.Run waits for its context and then spends the remaining deadline
	// as the shared hook budget, so it needs a context that is already done while
	// its deadline still carries the configured budget.
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), timeout)
	cancel()

	// Hooks run in reverse order: stop accepting requests, then release resources.
	shutdown.Run(shutdownCtx, log,
		func(context.Context) error {
			release()
			return nil
		},
		func(ctx context.Context) error { return server.Shutdown(ctx) },
	)

	return nil
}
