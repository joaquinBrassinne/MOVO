package cache_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/search"
	"github.com/joaquinBrassinne/MOVO/internal/search/adapters/cache"
)

func TestInvalidateMakesTheNextSearchHitTheDatabase(t *testing.T) {
	next := &countingFinder{out: []search.Branch{{BranchID: 1}}}
	store := newMapStore()
	f := newFinder(next, store)
	ctx := context.Background()

	for range 2 {
		if _, err := f.Nearby(ctx, newQuery()); err != nil {
			t.Fatal(err)
		}
	}
	if next.calls != 1 {
		t.Fatalf("before invalidation: %d database calls, want 1", next.calls)
	}

	if err := cache.Invalidate(ctx, store); err != nil {
		t.Fatalf("Invalidate: %v", err)
	}
	if _, err := f.Nearby(ctx, newQuery()); err != nil {
		t.Fatal(err)
	}
	if next.calls != 2 {
		t.Errorf("after invalidation: %d database calls, want 2", next.calls)
	}
	if _, err := f.Nearby(ctx, newQuery()); err != nil || next.calls != 2 {
		t.Errorf("the refreshed entry was not reused (calls %d, err %v)", next.calls, err)
	}
}

func TestAnUnreadableGenerationBypassesTheCache(t *testing.T) {
	next := &countingFinder{out: []search.Branch{{BranchID: 1}}}
	store := &mapStore{data: map[string][]byte{}, getErr: errors.New("down")}
	got, err := newFinder(next, store).Nearby(context.Background(), newQuery())
	if err != nil || len(got) != 1 || len(store.data) != 0 {
		t.Fatalf("got %v, %v with %d entries stored; want the database answer and no write", got, err, len(store.data))
	}
}

func generation(s *mapStore) string { return string(s.data["search:nearby:generation"]) }

func TestMiddlewareInvalidatesAfterASuccessfulWrite(t *testing.T) {
	for _, tc := range []struct {
		method, path string
		status       int
		want         bool
	}{
		{http.MethodPut, "/api/v1/me/business/branches/3/active", http.StatusNoContent, true},
		{http.MethodPost, "/api/v1/me/business/benefits", http.StatusCreated, true},
		{http.MethodDelete, "/api/v1/me/business/images/9", http.StatusNoContent, true},
		{http.MethodPost, "/api/v1/admin/businesses/4/actions", http.StatusOK, true},
		{http.MethodPut, "/api/v1/admin/reviews/2/status", http.StatusOK, true},
		{http.MethodPost, "/api/v1/access/tok/reviews", http.StatusCreated, true},
		// A handler that never calls WriteHeader answers 200.
		{http.MethodPut, "/api/v1/me/business", 0, true},
		// Failed writes changed nothing.
		{http.MethodPut, "/api/v1/me/business", http.StatusUnprocessableEntity, false},
		{http.MethodPut, "/api/v1/admin/subscriptions/1/plan", http.StatusInternalServerError, false},
		// Reads never invalidate.
		{http.MethodGet, "/api/v1/me/business", http.StatusOK, false},
		// Frequent or private writes that leave the search alone.
		{http.MethodPost, "/api/v1/me/business/codes/redeem", http.StatusOK, false},
		{http.MethodPost, "/api/v1/me/business/geocode", http.StatusOK, false},
		{http.MethodPost, "/api/v1/me/business/images/presign", http.StatusOK, false},
		{http.MethodPost, "/api/v1/benefits/5/codes", http.StatusCreated, false},
		{http.MethodPost, "/api/v1/businesses/slug/view", http.StatusNoContent, false},
		{http.MethodPost, "/api/v1/auth/login", http.StatusOK, false},
	} {
		store := newMapStore()
		h := cache.InvalidateOnWrite(store, slog.New(slog.NewTextHandler(io.Discard, nil)))(
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if tc.status != 0 {
					w.WriteHeader(tc.status)
				}
			}))
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(tc.method, tc.path, nil))

		if got := generation(store) == "1"; got != tc.want {
			t.Errorf("%s %s -> %d: invalidated = %v, want %v", tc.method, tc.path, tc.status, got, tc.want)
		}
	}
}

func TestMiddlewareSurvivesAFailingStore(t *testing.T) {
	store := &mapStore{data: map[string][]byte{}, setErr: errors.New("down")}
	h := cache.InvalidateOnWrite(store, slog.New(slog.NewTextHandler(io.Discard, nil)))(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusCreated) }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/me/business/branches", nil))
	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want the handler's 201 even if the cache is down", rec.Code)
	}
}
