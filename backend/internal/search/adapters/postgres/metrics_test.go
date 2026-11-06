package postgres_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
	"github.com/joaquinBrassinne/MOVO/internal/search"
	"github.com/joaquinBrassinne/MOVO/internal/search/adapters/postgres"
)

var march1 = time.Date(2026, 3, 1, 15, 0, 0, 0, time.UTC)

type metricsFixture struct {
	pool                      *pgxpool.Pool
	business, branch, service int64
}

func newMetricsFixture(t *testing.T) metricsFixture {
	t.Helper()

	s := &seeded{pool: dbtest.New(t)}
	branch := s.add(t, "yoga", "active", 100, false)
	f := metricsFixture{pool: s.pool, branch: branch}
	if err := s.pool.QueryRow(t.Context(), `SELECT business_id FROM branches WHERE id = $1`, branch).Scan(&f.business); err != nil {
		t.Fatalf("business of branch: %v", err)
	}
	if err := s.pool.QueryRow(t.Context(), `INSERT INTO services (business_id, name) VALUES ($1, 'Yoga') RETURNING id`, f.business).Scan(&f.service); err != nil {
		t.Fatalf("insert service: %v", err)
	}
	return f
}

// rows returns how many rows exist and the sum of their counts for a metric.
func (f metricsFixture) rows(t *testing.T, metric search.Metric) (n int, total int64) {
	t.Helper()
	if err := f.pool.QueryRow(t.Context(),
		`SELECT count(*), COALESCE(sum(count), 0) FROM metric_daily WHERE business_id = $1 AND metric = $2`,
		f.business, string(metric)).Scan(&n, &total); err != nil {
		t.Fatalf("read counters: %v", err)
	}
	return n, total
}

func TestIncrementAddsToTheSameRowWhenBranchAndServiceAreNull(t *testing.T) {
	t.Parallel()

	f := newMetricsFixture(t)
	counter := postgres.NewMetrics(f.pool)

	for range 3 {
		if err := counter.Increment(t.Context(), search.Event{Metric: search.ProfileView, BusinessID: f.business}, march1); err != nil {
			t.Fatalf("profile view: %v", err)
		}
	}
	if n, total := f.rows(t, search.ProfileView); n != 1 || total != 3 {
		t.Errorf("profile_view: %d rows, total %d; want 1 row and 3", n, total)
	}

	click := search.Event{Metric: search.WhatsAppClick, BusinessID: f.business, BranchID: f.branch}
	for range 2 {
		if err := counter.Increment(t.Context(), click, march1); err != nil {
			t.Fatalf("click without service: %v", err)
		}
	}
	if n, total := f.rows(t, search.WhatsAppClick); n != 1 || total != 2 {
		t.Errorf("whatsapp_click with NULL service: %d rows, total %d; want 1 row and 2", n, total)
	}
}

func TestIncrementKeepsEachDimensionApart(t *testing.T) {
	t.Parallel()

	f := newMetricsFixture(t)
	counter := postgres.NewMetrics(f.pool)
	click := search.Event{Metric: search.WhatsAppClick, BusinessID: f.business, BranchID: f.branch}
	clickOnService := click
	clickOnService.ServiceID = f.service

	for _, step := range []struct {
		event search.Event
		day   time.Time
	}{
		{click, march1},
		{click, march1.AddDate(0, 0, 1)}, // another day
		{clickOnService, march1},         // the same day, with a service
		{search.Event{Metric: search.ServiceView, BusinessID: f.business, BranchID: f.branch, ServiceID: f.service}, march1},
	} {
		if err := counter.Increment(t.Context(), step.event, step.day); err != nil {
			t.Fatalf("Increment %+v: %v", step.event, err)
		}
	}

	if n, total := f.rows(t, search.WhatsAppClick); n != 3 || total != 3 {
		t.Errorf("whatsapp_click: %d rows, total %d; want 3 rows (day, service) and 3", n, total)
	}
	if n, total := f.rows(t, search.ServiceView); n != 1 || total != 1 {
		t.Errorf("service_view: %d rows, total %d; want 1 and 1", n, total)
	}
}

// TestIncrementCountsEveryConcurrentRequest pins that the upsert is one atomic
// statement: no request is lost and no duplicate row appears.
func TestIncrementCountsEveryConcurrentRequest(t *testing.T) {
	t.Parallel()

	f := newMetricsFixture(t)
	counter := postgres.NewMetrics(f.pool)
	const requests = 40

	var wg sync.WaitGroup
	errs := make(chan error, requests)
	for range requests {
		wg.Go(func() {
			errs <- counter.Increment(t.Context(), search.Event{Metric: search.ProfileView, BusinessID: f.business}, march1)
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("Increment: %v", err)
		}
	}

	if n, total := f.rows(t, search.ProfileView); n != 1 || total != requests {
		t.Errorf("%d rows, total %d; want 1 row and %d", n, total, requests)
	}
}

func TestIncrementRejectsAnInvalidEventBeforeTheDatabase(t *testing.T) {
	t.Parallel()

	f := newMetricsFixture(t)
	counter := postgres.NewMetrics(f.pool)

	bad := search.Event{Metric: search.ProfileView, BusinessID: f.business, BranchID: f.branch}
	if err := counter.Increment(t.Context(), bad, march1); !errors.Is(err, search.ErrInvalidEvent) {
		t.Errorf("err = %v, want ErrInvalidEvent", err)
	}
	if n, _ := f.rows(t, search.ProfileView); n != 0 {
		t.Errorf("%d rows written for an invalid event, want none", n)
	}
}
