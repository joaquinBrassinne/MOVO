// Package commands holds the write use cases of the search module.
package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/search"
)

// argentina is the clock the days are counted on. The country has no daylight
// saving time, so a fixed offset is exact and needs no timezone database.
var argentina = time.FixedZone("ART", -3*60*60)

// botMarkers are lowercase fragments of the user agents of crawlers, link previews,
// uptime monitors and scripts. It is a basic filter on purpose (SDD 4.3 asks only
// that bots are ignored): a visitor whose agent happens to contain one is simply not
// counted, which is a negligible loss, while a bot that hides is not worth chasing.
var botMarkers = []string{
	"bot", "crawl", "spider", "slurp", "headless", "preview", "monitor", "uptime",
	"pingdom", "lighthouse", "facebookexternalhit", "whatsapp/", "curl/", "wget/",
	"python-requests", "go-http-client", "okhttp", "java/", "libwww", "httpclient",
}

// IsBot reports whether a user agent looks automated. An empty agent counts as a bot:
// browsers always send one.
func IsBot(userAgent string) bool {
	ua := strings.ToLower(strings.TrimSpace(userAgent))
	if ua == "" {
		return true
	}
	for _, marker := range botMarkers {
		if strings.Contains(ua, marker) {
			return true
		}
	}
	return false
}

// RecordMetric counts an event unless a bot caused it or the same visitor already caused
// the same event a moment ago. The user agent and the address are read to decide that and
// are not stored or passed on.
type RecordMetric struct {
	counter search.MetricCounter
	dedupe  search.Deduper
	now     func() time.Time
}

// NewRecordMetric builds the command. A nil clock uses time.Now.
func NewRecordMetric(counter search.MetricCounter, now func() time.Time) *RecordMetric {
	if now == nil {
		now = time.Now
	}
	return &RecordMetric{counter: counter, now: now}
}

// WithDeduper makes the command drop a repeated event of the same address (SDD 6.3b). An
// event with no address in its context is always counted.
func (r *RecordMetric) WithDeduper(d search.Deduper) *RecordMetric {
	r.dedupe = d
	return r
}

// Handle returns whether the event was counted. A bot or a repeat is not an error: it
// returns false and nil. An event with the wrong shape returns search.ErrInvalidEvent.
func (r *RecordMetric) Handle(ctx context.Context, e search.Event, userAgent string) (bool, error) {
	if err := e.Validate(); err != nil {
		return false, err
	}
	if IsBot(userAgent) {
		return false, nil
	}
	if address := search.ClientFrom(ctx); r.dedupe != nil && address != "" {
		key := fmt.Sprintf("%s|%d|%d|%d|%s", e.Metric, e.BusinessID, e.BranchID, e.ServiceID, address)
		duplicate, err := r.dedupe.Seen(ctx, key)
		if err != nil {
			return false, err
		}
		if duplicate {
			return false, nil
		}
	}
	if err := r.counter.Increment(ctx, e, r.now().In(argentina)); err != nil {
		return false, err
	}
	return true, nil
}
