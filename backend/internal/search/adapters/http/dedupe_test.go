package searchhttp_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
	"github.com/joaquinBrassinne/MOVO/internal/platform/storage"
	searchhttp "github.com/joaquinBrassinne/MOVO/internal/search/adapters/http"
	searchmemory "github.com/joaquinBrassinne/MOVO/internal/search/adapters/memory"
	searchpostgres "github.com/joaquinBrassinne/MOVO/internal/search/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/search/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/search/app/queries"
)

// newDedupingServer is newServer with the anti-inflation window on, as in main.
func newDedupingServer(t *testing.T) (http.Handler, *pgxpool.Pool) {
	t.Helper()

	pool := dbtest.New(t)
	branches := searchpostgres.NewBranches(pool)
	record := commands.NewRecordMetric(searchpostgres.NewMetrics(pool), nil).
		WithDeduper(searchmemory.NewDeduper(30*time.Minute, 100, nil))
	h := searchhttp.New(queries.NewNearbyBranches(branches), queries.NewFeaturedBranches(branches, nil), nil).
		WithWhatsApp(commands.NewOpenWhatsApp(branches, record, nil)).
		WithProfile(queries.NewBusinessProfile(branches, storage.NewMemory(), nil)).
		WithProfileView(commands.NewRecordProfileView(branches, record, nil))
	return httpx.NewRouter(nil, nil, h.Mount), pool
}

func requestFrom(h http.Handler, method, path, remoteAddr string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	req.RemoteAddr = remoteAddr
	req.Header.Set("User-Agent", browser)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestARepeatedViewOrClickFromTheSameAddressCountsOnce(t *testing.T) {
	t.Parallel()

	h, pool := newDedupingServer(t)
	branch := addBranch(t, pool, 1, 100)
	click := "/api/v1/r/wa/" + fmt.Sprint(branch)

	for range 3 {
		if rec := requestFrom(h, http.MethodPost, "/api/v1/businesses/studio-1/view", "198.51.100.1:5000"); rec.Code != http.StatusNoContent {
			t.Fatalf("view status = %d, want 204", rec.Code)
		}
		// The visitor always reaches WhatsApp, counted or not.
		if rec := requestFrom(h, http.MethodGet, click, "198.51.100.1:5000"); rec.Code != http.StatusFound {
			t.Fatalf("click status = %d, want 302", rec.Code)
		}
	}
	if total, _ := views(t, pool); total != 1 {
		t.Errorf("views = %d after three from one address, want 1", total)
	}
	if got := clicks(t, pool, branch); got != 1 {
		t.Errorf("clicks = %d after three from one address, want 1", got)
	}

	// Another address is another visitor, and the port does not make a new one.
	requestFrom(h, http.MethodPost, "/api/v1/businesses/studio-1/view", "198.51.100.2:5000")
	requestFrom(h, http.MethodPost, "/api/v1/businesses/studio-1/view", "198.51.100.2:6000")
	requestFrom(h, http.MethodGet, click, "198.51.100.2:5000")
	if total, _ := views(t, pool); total != 2 {
		t.Errorf("views = %d with a second address, want 2", total)
	}
	if got := clicks(t, pool, branch); got != 2 {
		t.Errorf("clicks = %d with a second address, want 2", got)
	}
}
