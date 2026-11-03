package searchhttp_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
	searchhttp "github.com/joaquinBrassinne/MOVO/internal/search/adapters/http"
	searchpostgres "github.com/joaquinBrassinne/MOVO/internal/search/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/search/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/search/app/queries"
)

func newServiceViewServer(t *testing.T) (http.Handler, *pgxpool.Pool) {
	t.Helper()

	pool := dbtest.New(t)
	branches := searchpostgres.NewBranches(pool)
	record := commands.NewRecordMetric(searchpostgres.NewMetrics(pool), nil)
	h := searchhttp.New(queries.NewNearbyBranches(branches), queries.NewFeaturedBranches(branches, nil), nil).
		WithServiceView(commands.NewRecordServiceView(branches, record, nil))
	return httpx.NewRouter(nil, nil, h.Mount), pool
}

func postServiceView(h http.Handler, branch, service, userAgent string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/branches/"+branch+"/services/"+service+"/view", nil)
	req.Header.Set("User-Agent", userAgent)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func serviceViews(t *testing.T, pool *pgxpool.Pool) (total, rows int64) {
	t.Helper()
	if err := pool.QueryRow(t.Context(),
		`SELECT COALESCE(sum(count), 0), count(*) FROM metric_daily WHERE metric = 'service_view'`).Scan(&total, &rows); err != nil {
		t.Fatalf("read service views: %v", err)
	}
	return total, rows
}

func TestServiceViewCountsOnTheBranchAndServiceShown(t *testing.T) {
	t.Parallel()

	h, pool := newServiceViewServer(t)
	branch := addBranch(t, pool, 1, 100)
	var service int64
	if err := pool.QueryRow(t.Context(),
		`INSERT INTO services (business_id, name) SELECT business_id, 'Yoga' FROM branches WHERE id = $1 RETURNING id`, branch).Scan(&service); err != nil {
		t.Fatal(err)
	}

	rec := postServiceView(h, fmt.Sprint(branch), fmt.Sprint(service), browser)
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("status = %d, body %q, cache %q; want 204, no body, no-store", rec.Code, rec.Body, rec.Header().Get("Cache-Control"))
	}
	if total, rows := serviceViews(t, pool); total != 1 || rows != 1 {
		t.Fatalf("after one view: total %d in %d rows, want 1 in 1", total, rows)
	}
	var gotBranch, gotService int64
	if err := pool.QueryRow(t.Context(),
		`SELECT branch_id, service_id FROM metric_daily WHERE metric = 'service_view'`).Scan(&gotBranch, &gotService); err != nil {
		t.Fatal(err)
	}
	if gotBranch != branch || gotService != service {
		t.Errorf("counted on branch %d service %d, want %d and %d", gotBranch, gotService, branch, service)
	}
}

func TestServiceViewAnswersTheSameWhenItCountsNothing(t *testing.T) {
	t.Parallel()

	h, pool := newServiceViewServer(t)
	branch := addBranch(t, pool, 1, 100)
	other := addBranch(t, pool, 2, 100)
	var active, inactive, foreign int64
	insert := func(dst *int64, sql string, args ...any) {
		t.Helper()
		if err := pool.QueryRow(t.Context(), sql, args...).Scan(dst); err != nil {
			t.Fatal(err)
		}
	}
	insert(&active, `INSERT INTO services (business_id, name) SELECT business_id, 'Yoga' FROM branches WHERE id = $1 RETURNING id`, branch)
	insert(&inactive, `INSERT INTO services (business_id, name, active) SELECT business_id, 'Pausado', false FROM branches WHERE id = $1 RETURNING id`, branch)
	insert(&foreign, `INSERT INTO services (business_id, name) SELECT business_id, 'Ajeno' FROM branches WHERE id = $1 RETURNING id`, other)

	for _, tt := range []struct {
		name, branch, service, agent string
	}{
		{"a bot", fmt.Sprint(branch), fmt.Sprint(active), "Googlebot/2.1"},
		{"an inactive service", fmt.Sprint(branch), fmt.Sprint(inactive), browser},
		{"a service of another business", fmt.Sprint(branch), fmt.Sprint(foreign), browser},
		{"a branch that does not exist", "999999", fmt.Sprint(active), browser},
		{"a service that does not exist", fmt.Sprint(branch), "999999", browser},
		{"ids that are not numbers", "abc", "xyz", browser},
		{"ids that are not positive", "0", "-1", browser},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rec := postServiceView(h, tt.branch, tt.service, tt.agent)
			if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
				t.Errorf("status = %d, body %q; want the same 204 with no body", rec.Code, rec.Body)
			}
			if total, _ := serviceViews(t, pool); total != 0 {
				t.Errorf("service views counted = %d, want 0", total)
			}
		})
	}
}
