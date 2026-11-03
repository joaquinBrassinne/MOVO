package main

import (
	"context"
	"log/slog"

	"github.com/joaquinBrassinne/MOVO/internal/platform/cache"
	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
	"github.com/joaquinBrassinne/MOVO/internal/search"
	searchcache "github.com/joaquinBrassinne/MOVO/internal/search/adapters/cache"
)

// newSearchCache builds the Redis cache of the nearby search (SDD 7.1). Without REDIS_URL it
// returns no decorators and the API runs as before; with it, an unreachable Redis stops the
// start-up so a typo in the URL is not mistaken for a working cache.
func newSearchCache(ctx context.Context, log *slog.Logger, cfg config.Cache) ([]func(search.BranchFinder) search.BranchFinder, func(), error) {
	if cfg.RedisURL == "" {
		return nil, func() {}, nil
	}
	store, err := cache.NewRedis(ctx, cfg.RedisURL)
	if err != nil {
		return nil, nil, err
	}
	log.Info("search cache enabled", "ttl", cfg.SearchTTL.String(), "cell_degrees", cfg.CellDegrees)
	wrap := func(next search.BranchFinder) search.BranchFinder {
		return searchcache.NewFinder(next, store, cfg.SearchTTL, cfg.CellDegrees, log)
	}
	return []func(search.BranchFinder) search.BranchFinder{wrap}, func() { _ = store.Close() }, nil
}
