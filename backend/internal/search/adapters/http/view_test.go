package searchhttp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func postView(h http.Handler, slug, userAgent string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/businesses/"+slug+"/view", nil)
	req.Header.Set("User-Agent", userAgent)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// views returns how many profile views are counted in total and in how many rows.
func views(t *testing.T, pool *pgxpool.Pool) (total, rows int64) {
	t.Helper()
	if err := pool.QueryRow(t.Context(),
		`SELECT COALESCE(sum(count), 0), count(*) FROM metric_daily WHERE metric = 'profile_view'`).Scan(&total, &rows); err != nil {
		t.Fatalf("read views: %v", err)
	}
	return total, rows
}

func TestProfileViewCountsOnceAndAddsToTheSameRow(t *testing.T) {
	t.Parallel()

	h, pool := newServer(t)
	addBranch(t, pool, 1, 100)

	rec := postView(h, "studio-1", browser)
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("status = %d, body %q; want 204 and no body", rec.Code, rec.Body)
	}
	if total, rows := views(t, pool); total != 1 || rows != 1 {
		t.Fatalf("after one view: total %d in %d rows, want 1 in 1", total, rows)
	}

	postView(h, "studio-1", browser)
	if total, rows := views(t, pool); total != 2 || rows != 1 {
		t.Errorf("after two views: total %d in %d rows, want 2 in the same row", total, rows)
	}

	var branch *int64
	if err := pool.QueryRow(t.Context(), `SELECT branch_id FROM metric_daily WHERE metric = 'profile_view'`).Scan(&branch); err != nil || branch != nil {
		t.Errorf("branch_id = %v, %v; want NULL", branch, err)
	}
}

func TestProfileViewSetsNoCookiesAndStoresNothingPersonal(t *testing.T) {
	t.Parallel()

	h, pool := newServer(t)
	addBranch(t, pool, 1, 100)

	rec := postView(h, "studio-1", browser)
	if cookies := rec.Header().Values("Set-Cookie"); len(cookies) != 0 {
		t.Errorf("Set-Cookie = %v, want none", cookies)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
}

func TestProfileViewIgnoresBotsAndBusinessesThatAreNotPublicWithTheSameAnswer(t *testing.T) {
	t.Parallel()

	h, pool := newServer(t)
	addBranch(t, pool, 1, 100)
	addBranch(t, pool, 2, 100)
	exec(t, pool, `UPDATE businesses SET status = 'paused' WHERE slug = 'studio-2'`)

	tests := []struct {
		name, slug, agent string
	}{
		{"a bot", "studio-1", "Googlebot/2.1"},
		{"no user agent", "studio-1", ""},
		{"a business that does not exist", "nobody", browser},
		{"a paused business", "studio-2", browser},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := postView(h, tt.slug, tt.agent)
			if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
				t.Errorf("status = %d, body %q; want the same 204 with no body", rec.Code, rec.Body)
			}
			if total, _ := views(t, pool); total != 0 {
				t.Errorf("views counted = %d, want 0", total)
			}
		})
	}
}
