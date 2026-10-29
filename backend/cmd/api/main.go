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

	"github.com/joaquinBrassinne/MOVO/internal/business/adapters/nominatim"
	identityhttp "github.com/joaquinBrassinne/MOVO/internal/identity/adapters/http"
	"github.com/joaquinBrassinne/MOVO/internal/identity/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/identity/adapters/ratelimit"
	"github.com/joaquinBrassinne/MOVO/internal/identity/adapters/tokens"
	"github.com/joaquinBrassinne/MOVO/internal/identity/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/platform/clientip"
	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
	"github.com/joaquinBrassinne/MOVO/internal/platform/db"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
	"github.com/joaquinBrassinne/MOVO/internal/platform/logger"
	"github.com/joaquinBrassinne/MOVO/internal/platform/shutdown"
	"github.com/joaquinBrassinne/MOVO/internal/platform/storage"
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

	plans, err := config.LoadPlans(os.Getenv)
	if err != nil {
		return err
	}

	storageCfg, err := config.LoadStorage(os.Getenv)
	if err != nil {
		return err
	}

	geocoding, err := config.LoadGeocoding(os.Getenv)
	if err != nil {
		return err
	}

	coupons, err := config.LoadCoupons(os.Getenv)
	if err != nil {
		return err
	}

	proxies, err := config.LoadProxies(os.Getenv)
	if err != nil {
		return err
	}

	signer, err := tokens.NewSigner(auth.JWTSecret, auth.AccessTokenTTL, nil)
	if err != nil {
		return err
	}

	pool, err := db.Connect(signalCtx, cfg.DatabaseURL)
	if err != nil {
		return err
	}

	images := storage.NewS3(storageCfg)
	if storageCfg.EnsureBucket {
		if err := ensureBucket(signalCtx, images); err != nil {
			pool.Close()
			return err
		}
	}

	businessMount, billingMount, err := newBusinessRoutes(log, pool, signer, plans, images, storageCfg.MaxImages, nominatim.New(nominatim.Config{URL: geocoding.URL, UserAgent: geocoding.UserAgent}))
	if err != nil {
		pool.Close()
		return err
	}

	if err := seedSuperAdmin(signalCtx, log, pool, auth); err != nil {
		pool.Close()
		return err
	}

	router := httpx.NewRouter(log, pool, newIdentityRoutes(log, pool, auth, signer), businessMount, billingMount, newSearchRoutes(log, pool, images), newBenefitsRoutes(log, pool, signer, coupons.MaxAttempts), newReviewsRoutes(log, pool, signer))
	// The client address feeds the login and coupon limits, so it is resolved before any route.
	handler := clientip.Middleware(proxies.Trusted)(router)
	return serve(signalCtx, log, cfg.HTTPAddr, cfg.ShutdownTimeout, handler, pool.Close)
}

// ensureBucket creates the development bucket when it is missing. It is bounded
// so an unreachable object server stops the start-up instead of hanging it.
func ensureBucket(ctx context.Context, images *storage.S3) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := images.EnsureBucket(ctx); err != nil {
		return fmt.Errorf("prepare object storage: %w", err)
	}
	return nil
}

// seedSuperAdmin creates the first super admin when the environment configures
// one. A configuration that cannot be honoured stops the process at startup.
func seedSuperAdmin(ctx context.Context, log *slog.Logger, pool *pgxpool.Pool, auth config.Auth) error {
	if auth.SuperAdminEmail == "" {
		return nil
	}

	created, err := commands.NewEnsureSuperAdmin(postgres.NewUsers(pool)).Handle(ctx, auth.SuperAdminEmail, auth.SuperAdminPassword)
	if err != nil {
		return fmt.Errorf("seed super admin: %w", err)
	}
	if created {
		log.Info("super admin created")
	}

	return nil
}

// newIdentityRoutes composes the identity module: adapters, use cases and HTTP.
func newIdentityRoutes(log *slog.Logger, pool *pgxpool.Pool, auth config.Auth, signer *tokens.Signer) func(chi.Router) {
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
	).Mount
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
