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
	latCell, lngCell := cellIndex(q.Lat, f.cell), cellIndex(q.Lng, f.cell)
	key := fmt.Sprintf("%s:%d:%d:%d:%s:%g:%d:%d", keyPrefix, latCell, lngCell, q.RadiusM, q.Category, q.MinRating, q.Limit, q.Offset)

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

func cellIndex(v, cell float64) int64 { return int64(math.Floor(v / cell)) }

// cellCenter is the middle of cell idx, kept inside [-limit, limit] so it stays a valid point.
func cellCenter(idx int64, cell, limit float64) float64 {
	return math.Max(-limit, math.Min(limit, (float64(idx)+0.5)*cell))
}
