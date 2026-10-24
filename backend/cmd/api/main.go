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

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	identityhttp "github.com/joaquinBrassinne/MOVO/internal/identity/adapters/http"
	"github.com/joaquinBrassinne/MOVO/internal/identity/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/identity/adapters/ratelimit"
	"github.com/joaquinBrassinne/MOVO/internal/identity/adapters/tokens"
	"github.com/joaquinBrassinne/MOVO/internal/identity/app/commands"
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

// Failed-login budget: after loginMaxAttempts failures within loginWindow an
// email or a client address is blocked until the window ends.
const (
	loginMaxAttempts = 5
	loginWindow      = 15 * time.Minute
	loginMaxKeys     = 100_000
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

	auth, err := config.LoadAuth(os.Getenv)
	if err != nil {
		return err
	}

	pool, err := db.Connect(signalCtx, cfg.DatabaseURL)
	if err != nil {
		return err
	}

	identityRoutes, err := newIdentityRoutes(log, pool, auth)
	if err != nil {
		pool.Close()
		return err
	}

	return serve(signalCtx, log, cfg.HTTPAddr, cfg.ShutdownTimeout, httpx.NewRouter(log, pool, identityRoutes), pool.Close)
}

// newIdentityRoutes composes the identity module: adapters, use cases and HTTP.
func newIdentityRoutes(log *slog.Logger, pool *pgxpool.Pool, auth config.Auth) (func(chi.Router), error) {
	signer, err := tokens.NewSigner(auth.JWTSecret, auth.AccessTokenTTL, nil)
	if err != nil {
		return nil, err
	}

	users := postgres.NewUsers(pool)
	store := postgres.NewRefreshTokens(pool)
	limiter := ratelimit.New(ratelimit.Config{
		MaxAttempts: loginMaxAttempts,
		Window:      loginWindow,
		MaxKeys:     loginMaxKeys,
	}, nil)
	sessions := commands.NewSessions(signer, store, users, auth.RefreshTokenTTL, nil, log)

	return identityhttp.New(
		commands.NewRegister(users, sessions),
		commands.NewLogin(users, limiter, sessions),
		commands.NewRefresh(sessions),
		commands.NewLogout(store, nil),
		log,
	).Mount, nil
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
