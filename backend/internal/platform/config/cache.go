package config

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Environment variables understood by LoadCache (SDD 7.1).
const (
	envRedisURL         = "REDIS_URL"
	envSearchCacheTTL   = "SEARCH_CACHE_TTL"
	envSearchCacheCellD = "SEARCH_CACHE_CELL_DEG"
)

// Defaults of the search cache: a minute keeps a result fresh enough for a map, and a cell
// of 0.002 degrees (about 220 m) is far below the 2 km distance band of the ranking.
const (
	defaultSearchCacheTTL   = time.Minute
	defaultSearchCacheCellD = 0.002
	maxSearchCacheTTL       = time.Hour
	maxSearchCacheCellD     = 0.05
)

// Cache holds the settings of the Redis search cache. Redis is optional: without RedisURL the
// API runs exactly as before.
type Cache struct {
	// RedisURL is the Redis connection string; empty turns the cache off.
	RedisURL string
	// SearchTTL is how long a cached search result lives.
	SearchTTL time.Duration
	// CellDegrees is the side, in degrees, of the grid the search origin is snapped to.
	CellDegrees float64
}

// LoadCache builds Cache from get, normally os.Getenv. Unset values keep the defaults;
// anything invalid is refused at start-up.
func LoadCache(get func(string) string) (Cache, error) {
	if get == nil {
		return Cache{}, errors.New("config: environment lookup must not be nil")
	}

	out := Cache{
		RedisURL:    strings.TrimSpace(get(envRedisURL)),
		SearchTTL:   defaultSearchCacheTTL,
		CellDegrees: defaultSearchCacheCellD,
	}
	if raw := strings.TrimSpace(get(envSearchCacheTTL)); raw != "" {
		ttl, err := time.ParseDuration(raw)
		if err != nil || ttl <= 0 || ttl > maxSearchCacheTTL {
			return Cache{}, fmt.Errorf("config: %s %q must be a duration above zero and up to %s", envSearchCacheTTL, raw, maxSearchCacheTTL)
		}
		out.SearchTTL = ttl
	}
	if raw := strings.TrimSpace(get(envSearchCacheCellD)); raw != "" {
		cell, err := strconv.ParseFloat(raw, 64)
		if err != nil || math.IsNaN(cell) || cell <= 0 || cell > maxSearchCacheCellD {
			return Cache{}, fmt.Errorf("config: %s %q must be a number above 0 and up to %v", envSearchCacheCellD, raw, maxSearchCacheCellD)
		}
		out.CellDegrees = cell
	}
	return out, nil
}
