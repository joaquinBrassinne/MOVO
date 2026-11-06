package searchhttp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
	"github.com/joaquinBrassinne/MOVO/internal/search"
	searchhttp "github.com/joaquinBrassinne/MOVO/internal/search/adapters/http"
	"github.com/joaquinBrassinne/MOVO/internal/search/app/queries"
)

// emptyFinder answers every search with no branches, so these tests need no database.
type emptyFinder struct{}

func (emptyFinder) Nearby(context.Context, search.Query) ([]search.Branch, error) { return nil, nil }

func (emptyFinder) Featured(context.Context, search.Query, int64) ([]search.Branch, error) {
	return nil, nil
}

// budgetLimiter allows n requests per key and then refuses with a fixed wait.
type budgetLimiter struct {
	n     int
	seen  map[string]int
	retry time.Duration
}

func (b *budgetLimiter) Allow(key string) (time.Duration, bool) {
	b.seen[key]++
	if b.seen[key] > b.n {
		return b.retry, false
	}
	return 0, true
}

func newLimited(l searchhttp.Limiter) http.Handler {
	finder := emptyFinder{}
	h := searchhttp.New(queries.NewNearbyBranches(finder), queries.NewFeaturedBranches(finder, nil), nil).WithRateLimit(l)
	return httpx.NewRouter(nil, nil, h.Mount)
}

func getFrom(h http.Handler, path, remote string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.RemoteAddr = remote
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestSearchIsRefusedOnceTheBudgetIsSpent(t *testing.T) {
	h := newLimited(&budgetLimiter{n: 2, seen: map[string]int{}, retry: 1500 * time.Millisecond})
	path := "/api/v1/search?" + origin

	for i := range 2 {
		if rec := getFrom(h, path, "203.0.113.7:5000"); rec.Code != http.StatusOK {
			t.Fatalf("request %d = %d, want 200 while inside the budget", i+1, rec.Code)
		}
	}
	rec := getFrom(h, path, "203.0.113.7:5000")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("third request = %d, want 429", rec.Code)
	}
	if got := rec.Header().Get("Retry-After"); got != "2" {
		t.Errorf("Retry-After = %q, want 2 (1.5 s rounded up)", got)
	}
	var body struct {
		Error struct {
			Code    string         `json:"code"`
			Details map[string]int `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not the error envelope: %v", err)
	}
	if body.Error.Code != searchhttp.CodeRateLimited || body.Error.Details["retry_after_seconds"] != 2 {
		t.Errorf("error = %+v, want rate_limited with retry_after_seconds 2", body.Error)
	}
}

func TestFeaturedSharesTheBudgetOfSearch(t *testing.T) {
	h := newLimited(&budgetLimiter{n: 1, seen: map[string]int{}, retry: time.Second})
	if rec := getFrom(h, "/api/v1/search?"+origin, "203.0.113.7:5000"); rec.Code != http.StatusOK {
		t.Fatalf("search = %d, want 200", rec.Code)
	}
	if rec := getFrom(h, "/api/v1/home/featured?"+origin+"&radius_m=5000&limit=5", "203.0.113.7:5000"); rec.Code != http.StatusTooManyRequests {
		t.Errorf("featured after the budget = %d, want 429", rec.Code)
	}
}

func TestEachClientAddressHasItsOwnBudget(t *testing.T) {
	h := newLimited(&budgetLimiter{n: 1, seen: map[string]int{}, retry: time.Second})
	path := "/api/v1/search?" + origin
	if rec := getFrom(h, path, "203.0.113.7:5000"); rec.Code != http.StatusOK {
		t.Fatalf("first client = %d, want 200", rec.Code)
	}
	if rec := getFrom(h, path, "203.0.113.7:5000"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("first client again = %d, want 429", rec.Code)
	}
	if rec := getFrom(h, path, "198.51.100.9:5000"); rec.Code != http.StatusOK {
		t.Errorf("second client = %d, want 200: one address must not spend another's budget", rec.Code)
	}
}

func TestWithoutALimiterSearchIsUnlimited(t *testing.T) {
	h := newLimited(nil)
	for i := range 50 {
		if rec := getFrom(h, "/api/v1/search?"+origin, "203.0.113.7:5000"); rec.Code != http.StatusOK {
			t.Fatalf("request %d = %d, want 200", i+1, rec.Code)
		}
	}
}
