package main

import (
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/adapters/ratelimit"
	"github.com/joaquinBrassinne/MOVO/internal/platform/cache"
	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
	"github.com/joaquinBrassinne/MOVO/internal/search"
	searchhttp "github.com/joaquinBrassinne/MOVO/internal/search/adapters/http"
	searchmemory "github.com/joaquinBrassinne/MOVO/internal/search/adapters/memory"
	searchpostgres "github.com/joaquinBrassinne/MOVO/internal/search/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/search/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/search/app/queries"
)

// Anti-inflation of the metrics: a visit or click repeated from the same address within the
// window is not counted again. Thirty minutes ignores refreshes and bounces but counts the
// person who comes back later; the table is bounded so a flood cannot grow the memory.
const (
	metricDedupeWindow  = 30 * time.Minute
	metricDedupeMaxKeys = 100_000
)

// searchRateMaxKeys bounds the memory of the in-memory search limit: one key per client address.
const searchRateMaxKeys = 100_000

// newSearchLimiter builds the per-address request limit of the public search (SDD 7.3). It
// reuses the fixed-window counters of the coupon limit: in memory, or in Redis when a store is
// given so every API instance shares the count. A budget of 0 turns it off (nil).
func newSearchLimiter(log *slog.Logger, rate config.SearchRate, store *cache.Redis) searchhttp.Limiter {
	if rate.PerMinute == 0 {
		return nil
	}
	limit := ratelimit.Config{MaxAttempts: rate.PerMinute, Window: time.Minute, MaxKeys: searchRateMaxKeys}
	if store == nil {
		return ratelimit.New(limit, nil)
	}
	return ratelimit.NewRedis(store, "search", limit, log)
}

// newSearchRoutes composes the search module: adapters, use cases and HTTP. limiter caps the
// public search per client address (nil leaves it unlimited) and decorate optionally wraps the
// nearby search, as the Redis cache does (SDD 7.1).
func newSearchRoutes(log *slog.Logger, pool *pgxpool.Pool, images search.ImageURLs, limiter searchhttp.Limiter, decorate ...func(search.BranchFinder) search.BranchFinder) func(chi.Router) {
	branches := searchpostgres.NewBranches(pool)
	var nearby search.BranchFinder = branches
	for _, wrap := range decorate {
		nearby = wrap(nearby)
	}
	// The same visitor repeating an event within the window counts once (SDD 6.3b).
	record := commands.NewRecordMetric(searchpostgres.NewMetrics(pool), nil).
		WithDeduper(searchmemory.NewDeduper(metricDedupeWindow, metricDedupeMaxKeys, nil))
	return searchhttp.New(queries.NewNearbyBranches(nearby), queries.NewFeaturedBranches(branches, nil), log).
		WithRateLimit(limiter).
		WithWhatsApp(commands.NewOpenWhatsApp(branches, record, log)).
		WithProfile(queries.NewBusinessProfile(branches, images, nil)).
		WithProfileView(commands.NewRecordProfileView(branches, record, log)).
		WithServiceView(commands.NewRecordServiceView(branches, record, log)).Mount
}
