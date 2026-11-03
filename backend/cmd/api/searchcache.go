package main

import (
	"log/slog"
	"net/http"

	"github.com/joaquinBrassinne/MOVO/internal/platform/cache"
	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
	"github.com/joaquinBrassinne/MOVO/internal/search"
	searchcache "github.com/joaquinBrassinne/MOVO/internal/search/adapters/cache"
)

// searchCache is the Redis cache of the nearby search (SDD 7.1 and 7.2). Its zero value is
// "off": no decorators, no middleware.
type searchCache struct {
	decorators []func(search.BranchFinder) search.BranchFinder
	// invalidate drops the cache after writes that change what the search shows.
	invalidate func(http.Handler) http.Handler
}

// newSearchCache builds the cache on the Redis connection shared with the rate limits. A nil
// store (REDIS_URL unset) leaves it off and the API runs as before.
func newSearchCache(log *slog.Logger, cfg config.Cache, store *cache.Redis) searchCache {
	if store == nil {
		return searchCache{invalidate: func(next http.Handler) http.Handler { return next }}
	}
	log.Info("search cache enabled", "ttl", cfg.SearchTTL.String(), "cell_degrees", cfg.CellDegrees)
	wrap := func(next search.BranchFinder) search.BranchFinder {
		return searchcache.NewFinder(next, store, cfg.SearchTTL, cfg.CellDegrees, log)
	}
	return searchCache{
		decorators: []func(search.BranchFinder) search.BranchFinder{wrap},
		invalidate: searchcache.InvalidateOnWrite(store, log),
	}
}
