package main

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/joaquinBrassinne/MOVO/internal/platform/cache"
	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
	"github.com/joaquinBrassinne/MOVO/internal/search"
	searchcache "github.com/joaquinBrassinne/MOVO/internal/search/adapters/cache"
)

// searchCache is the Redis cache of the nearby search (SDD 7.1 and 7.2). Its zero value is
// "off": no decorators, no middleware, nothing to close.
type searchCache struct {
	decorators []func(search.BranchFinder) search.BranchFinder
	// invalidate drops the cache after writes that change what the search shows.
	invalidate func(http.Handler) http.Handler
	close      func()
}

// newSearchCache builds the cache. Without REDIS_URL it is off and the API runs as before;
// with it, an unreachable Redis stops the start-up so a typo in the URL is not mistaken for
// a working cache.
func newSearchCache(ctx context.Context, log *slog.Logger, cfg config.Cache) (searchCache, error) {
	if cfg.RedisURL == "" {
		return searchCache{invalidate: func(next http.Handler) http.Handler { return next }, close: func() {}}, nil
	}
	store, err := cache.NewRedis(ctx, cfg.RedisURL)
	if err != nil {
		return searchCache{}, err
	}
	log.Info("search cache enabled", "ttl", cfg.SearchTTL.String(), "cell_degrees", cfg.CellDegrees)
	wrap := func(next search.BranchFinder) search.BranchFinder {
		return searchcache.NewFinder(next, store, cfg.SearchTTL, cfg.CellDegrees, log)
	}
	return searchCache{
		decorators: []func(search.BranchFinder) search.BranchFinder{wrap},
		invalidate: searchcache.InvalidateOnWrite(store, log),
		close:      func() { _ = store.Close() },
	}, nil
}
