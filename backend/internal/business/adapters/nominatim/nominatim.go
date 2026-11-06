// Package nominatim implements business.Geocoder over a Nominatim server, by
// default the public one. Its usage policy sets the shape of this adapter: one
// request per second for the whole process, a User-Agent that identifies the
// application, results cached, no autocomplete (see docs/sdd.md §12.3).
package nominatim

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/business"
)

const (
	defaultInterval = time.Second
	defaultMaxWait  = 3 * time.Second
	requestTimeout  = 8 * time.Second
	cacheTTL        = 30 * 24 * time.Hour
	cacheCapacity   = 2000
	maxResponseSize = 64 << 10
)

// Config configures the geocoder. Interval and MaxWait have defaults that follow
// the public server policy; tests shorten them.
type Config struct {
	// URL is the server, without trailing slash.
	URL string
	// UserAgent identifies the application to the server.
	UserAgent string
	// Interval is the minimum time between two requests to the server.
	Interval time.Duration
	// MaxWait is how long a request may queue for its turn before it is refused.
	MaxWait time.Duration
}

// Geocoder implements business.Geocoder.
type Geocoder struct {
	cfg    Config
	client *http.Client

	gate  sync.Mutex // guards next
	next  time.Time  // the earliest moment the next request may be sent
	cache *cache
}

// New builds the geocoder.
func New(cfg Config) *Geocoder {
	if cfg.Interval <= 0 {
		cfg.Interval = defaultInterval
	}
	if cfg.MaxWait <= 0 {
		cfg.MaxWait = defaultMaxWait
	}
	return &Geocoder{cfg: cfg, client: &http.Client{Timeout: requestTimeout}, cache: newCache(cacheCapacity, cacheTTL)}
}

type result struct {
	Lat         string `json:"lat"`
	Lon         string `json:"lon"`
	DisplayName string `json:"display_name"`
}

// Geocode returns the best match for the query, restricted to Argentina. Repeated
// queries (ignoring case and spacing) are answered from the cache, misses included.
func (g *Geocoder) Geocode(ctx context.Context, query string) (business.Place, error) {
	key := normalize(query)
	if entry, ok := g.cache.get(key); ok {
		return entry.place, entry.err
	}

	if err := g.waitForTurn(ctx); err != nil {
		return business.Place{}, err
	}

	place, err := g.fetch(ctx, query)
	// Only definitive answers are remembered: a failure may be temporary.
	if err == nil || errors.Is(err, business.ErrAddressNotFound) {
		g.cache.put(key, cached{place: place, err: err})
	}
	return place, err
}

// waitForTurn reserves the next request slot and sleeps until it arrives. When the
// queue is longer than MaxWait it refuses at once and reserves nothing, so a burst
// cannot push the whole service into a long line.
func (g *Geocoder) waitForTurn(ctx context.Context) error {
	g.gate.Lock()
	now := time.Now()
	slot := g.next
	if slot.Before(now) {
		slot = now
	}
	if slot.Sub(now) > g.cfg.MaxWait {
		g.gate.Unlock()
		return business.ErrGeocoderBusy
	}
	g.next = slot.Add(g.cfg.Interval)
	g.gate.Unlock()

	timer := time.NewTimer(slot.Sub(now))
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (g *Geocoder) fetch(ctx context.Context, query string) (business.Place, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("format", "jsonv2")
	params.Set("limit", "1")
	params.Set("countrycodes", "ar")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.cfg.URL+"/search?"+params.Encode(), nil)
	if err != nil {
		return business.Place{}, fmt.Errorf("%w: build request: %w", business.ErrGeocoderUnavailable, err)
	}
	req.Header.Set("User-Agent", g.cfg.UserAgent)
	req.Header.Set("Accept", "application/json")

	res, err := g.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return business.Place{}, ctx.Err()
		}
		return business.Place{}, fmt.Errorf("%w: %w", business.ErrGeocoderUnavailable, err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return business.Place{}, fmt.Errorf("%w: status %d", business.ErrGeocoderUnavailable, res.StatusCode)
	}

	var results []result
	if err := json.NewDecoder(io.LimitReader(res.Body, maxResponseSize)).Decode(&results); err != nil {
		return business.Place{}, fmt.Errorf("%w: decode answer: %w", business.ErrGeocoderUnavailable, err)
	}
	if len(results) == 0 {
		return business.Place{}, business.ErrAddressNotFound
	}
	return toPlace(results[0])
}

func toPlace(r result) (business.Place, error) {
	lat, errLat := strconv.ParseFloat(r.Lat, 64)
	lon, errLon := strconv.ParseFloat(r.Lon, 64)
	if errLat != nil || errLon != nil || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return business.Place{}, fmt.Errorf("%w: unusable coordinates", business.ErrGeocoderUnavailable)
	}
	return business.Place{Latitude: lat, Longitude: lon, DisplayName: r.DisplayName}, nil
}

// normalize makes equal addresses share a cache entry: lowercase, single spaces.
func normalize(query string) string {
	return strings.Join(strings.Fields(strings.ToLower(query)), " ")
}
