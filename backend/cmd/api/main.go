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

	apispec "github.com/joaquinBrassinne/MOVO/api"
	"github.com/joaquinBrassinne/MOVO/internal/business/adapters/nominatim"
	"github.com/joaquinBrassinne/MOVO/internal/identity"
	identityhttp "github.com/joaquinBrassinne/MOVO/internal/identity/adapters/http"
	"github.com/joaquinBrassinne/MOVO/internal/identity/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/identity/adapters/ratelimit"
	"github.com/joaquinBrassinne/MOVO/internal/identity/adapters/tokens"
	"github.com/joaquinBrassinne/MOVO/internal/identity/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/platform/apidocs"
	"github.com/joaquinBrassinne/MOVO/internal/platform/cache"
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

// Sign-up budget: one address may try registerMaxAttempts sign-ups per
// registerWindow, accepted or not. An owner registers once, so a shared office
// address has plenty of room.
const (
	registerMaxAttempts = 10
	registerWindow      = time.Hour
	registerMaxKeys     = 100_000
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

	cacheCfg, err := config.LoadCache(os.Getenv)
	if err != nil {
		return err
	}

	signer, err := tokens.NewSigner(auth.JWTSecret, auth.AccessTokenTTL, nil)
	if err != nil {
		return err
	}

	searchRate, err := config.LoadSearchRate(os.Getenv)
	if err != nil {
		return err
	}

	poolCfg, err := config.LoadPool(os.Getenv)
	if err != nil {
		return err
	}

	pool, err := db.ConnectWithPool(signalCtx, cfg.DatabaseURL, poolCfg)
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

	metricsAccess, err := newMetricsPlan(pool, plans)
	if err != nil {
		pool.Close()
		return err
	}

	if err := seedSuperAdmin(signalCtx, log, pool, auth); err != nil {
		pool.Close()
		return err
	}

	// One Redis connection serves the search cache and the rate limits; nil when REDIS_URL is unset.
	redisStore, err := openRedis(signalCtx, cacheCfg)
	if err != nil {
		pool.Close()
		return err
	}
	closeRedis := func() {
		if redisStore != nil {
			_ = redisStore.Close()
		}
	}
	if redisStore != nil {
		log.Info("rate limits shared through redis")
	}
	searchCache := newSearchCache(log, cacheCfg, redisStore)

	router := httpx.NewRouter(log, pool, newIdentityRoutes(log, pool, auth, signer, redisStore), businessMount, billingMount, newSearchRoutes(log, pool, images, newSearchLimiter(log, searchRate, redisStore), searchCache.decorators...), newBenefitsRoutes(log, pool, signer, coupons.MaxAttempts, metricsAccess, redisStore), newReviewsRoutes(log, pool, signer), newAdminRoutes(log, pool, signer))
	// The client address feeds the login and coupon limits, so it is resolved before any route.
	handler := searchCache.invalidate(clientip.Middleware(proxies.Trusted)(router))
	if cfg.APIDocs {
		handler = apidocs.Mount(apispec.Spec, handler)
		log.Info("api docs enabled", "path", apidocs.PagePath)
	}
	return serve(signalCtx, log, cfg.HTTPAddr, cfg.ShutdownTimeout, handler, func() { closeRedis(); pool.Close() })
}

// openRedis connects to REDIS_URL, or returns nil when it is unset. An unreachable Redis stops
// the start-up, so a typo in the URL is not mistaken for working shared limits.
func openRedis(ctx context.Context, cfg config.Cache) (*cache.Redis, error) {
	if cfg.RedisURL == "" {
		return nil, nil
	}
	return cache.NewRedis(ctx, cfg.RedisURL)
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

// newIdentityRoutes composes the identity module: adapters, use cases and HTTP. With a Redis
// store the login and sign-up limits are counted in Redis, shared by every instance; nil keeps
// them in memory.
func newIdentityRoutes(log *slog.Logger, pool *pgxpool.Pool, auth config.Auth, signer *tokens.Signer, redisStore *cache.Redis) func(chi.Router) {
	users := postgres.NewUsers(pool)
	store := postgres.NewRefreshTokens(pool)
	loginLimit := ratelimit.Config{
		MaxAttempts: loginMaxAttempts,
		Window:      loginWindow,
		MaxKeys:     loginMaxKeys,
	}
	// Its own budget and its own keys: a burst of sign-ups must not block anyone's login.
	signUpLimit := ratelimit.Config{
		MaxAttempts: registerMaxAttempts,
		Window:      registerWindow,
		MaxKeys:     registerMaxKeys,
	}
	var (
		limiter identity.LoginLimiter
		signUps identity.RegistrationLimiter
	)
	if redisStore == nil {
		limiter = ratelimit.New(loginLimit, nil)
		signUps = ratelimit.New(signUpLimit, nil)
	} else {
		limiter = ratelimit.NewRedis(redisStore, "login", loginLimit, log)
		signUps = ratelimit.NewRedis(redisStore, "signup", signUpLimit, log)
	}
	sessions := commands.NewSessions(signer, store, users, auth.RefreshTokenTTL, nil, log)

	handlers := identityhttp.New(
		commands.NewRegister(users, signUps, sessions),
		commands.NewLogin(users, limiter, sessions),
		commands.NewRefresh(sessions),
		commands.NewLogout(store, nil),
		log,
	)
	// The reset link's redeem counts against the same per-address budget as sign-ups, under
	// its own key, so a burst of guesses cannot block anyone's login.
	resets := postgres.NewPasswordResets(pool)
	reset := identityhttp.NewResetHandlers(
		commands.NewIssuePasswordReset(users, resets, nil),
		commands.NewResetPassword(resets, signUps, nil),
		signer,
		log,
	)

	return func(r chi.Router) {
		handlers.Mount(r)
		reset.Mount(r)
	}
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
