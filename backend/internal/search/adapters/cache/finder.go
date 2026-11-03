// Package cache decorates the branch search with a cache-aside layer (SDD 7.1). The cache is
// not part of the domain: it wraps a search.BranchFinder and can be removed without touching it.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/search"
)

// Store is the key-value store the decorator needs. Get reports found=false on a miss.
type Store interface {
	Get(ctx context.Context, key string) (value []byte, found bool, err error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	// Incr adds one to the counter at key, creating it at zero first.
	Incr(ctx context.Context, key string) error
}

// generationKey holds the cache generation. Every entry key carries it, so bumping it makes
// all earlier entries unreachable at once (they expire on their own TTL); that is the
// invalidation of SDD 7.2, one write instead of a scan over the cells that changed.
const generationKey = "search:nearby:generation"

// Invalidate drops every cached search. A store error is returned for the caller to log:
// the TTL bounds how long an entry can then stay stale.
func Invalidate(ctx context.Context, store Store) error {
	return store.Incr(ctx, generationKey)
}

// keyPrefix versions the cached shape: bump it when search.Branch changes so old entries are
// never decoded into the new struct.
const keyPrefix = "search:nearby:v1"

// Finder serves Nearby from the store and falls back to the wrapped finder on a miss. The
// origin is snapped to a grid cell so nearby users share one entry; the search then runs from
// the center of the cell, which keeps a cached answer identical to a fresh one for its key.
// A store failure never fails a search: it is logged and the database answers.
type Finder struct {
	next  search.BranchFinder
	store Store
	ttl   time.Duration
	cell  float64
	log   *slog.Logger
}

// NewFinder wraps next. ttl is how long an entry lives and cellDegrees the grid side.
func NewFinder(next search.BranchFinder, store Store, ttl time.Duration, cellDegrees float64, log *slog.Logger) *Finder {
	if log == nil {
		log = slog.Default()
	}
	return &Finder{next: next, store: store, ttl: ttl, cell: cellDegrees, log: log}
}

// Nearby implements search.BranchFinder.
func (f *Finder) Nearby(ctx context.Context, q search.Query) ([]search.Branch, error) {
	if q.Validate() != nil {
		return f.next.Nearby(ctx, q) // invalid input is never cached; the finder reports it
	}
	// The generation is read before the database so an invalidation that lands while the
	// query runs leaves the new entry under the old generation, where nobody reads it.
	gen, err := f.generation(ctx)
	if err != nil {
		f.log.WarnContext(ctx, "search cache generation unreadable", "error", err)
		return f.next.Nearby(ctx, q)
	}
	latCell, lngCell := cellIndex(q.Lat, f.cell), cellIndex(q.Lng, f.cell)
	key := fmt.Sprintf("%s:g%s:%d:%d:%d:%s:%g:%d:%d", keyPrefix, gen, latCell, lngCell, q.RadiusM, q.Category, q.MinRating, q.Limit, q.Offset)

	if raw, found, err := f.store.Get(ctx, key); err != nil {
		f.log.WarnContext(ctx, "search cache read failed", "error", err)
	} else if found {
		var cached []search.Branch
		if err := json.Unmarshal(raw, &cached); err == nil {
			return cached, nil
		}
		f.log.WarnContext(ctx, "search cache entry unreadable", "key", key)
	}

	snapped := q
	snapped.Lat, snapped.Lng = cellCenter(latCell, f.cell, 90), cellCenter(lngCell, f.cell, 180)
	items, err := f.next.Nearby(ctx, snapped)
	if err != nil {
		return nil, err
	}

	if raw, err := json.Marshal(items); err == nil {
		if err := f.store.Set(ctx, key, raw, f.ttl); err != nil {
			f.log.WarnContext(ctx, "search cache write failed", "error", err)
		}
	}
	return items, nil
}

// generation is the current cache generation; a store that has none yet is at "0".
func (f *Finder) generation(ctx context.Context) (string, error) {
	raw, found, err := f.store.Get(ctx, generationKey)
	if err != nil {
		return "", err
	}
	if !found {
		return "0", nil
	}
	return string(raw), nil
}

func cellIndex(v, cell float64) int64 { return int64(math.Floor(v / cell)) }

// cellCenter is the middle of cell idx, kept inside [-limit, limit] so it stays a valid point.
func cellCenter(idx int64, cell, limit float64) float64 {
	return math.Max(-limit, math.Min(limit, (float64(idx)+0.5)*cell))
}
