//go:build perf

package postgres

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
	"github.com/joaquinBrassinne/MOVO/internal/search"
)

// Search budget from the SDD (phase 3 DoD): p95 under 300 ms over 10k businesses.
const (
	seededBusinesses = 10_000
	budget           = 300 * time.Millisecond
	samples          = 400
)

// seedBusinesses inserts n fake businesses with one branch each. Sixty percent sit
// inside a metro area around Rio Tercero, the rest spread over the country; ten
// percent are paused, five percent of the branches are inactive and five percent of
// the businesses are pilots (the stand-in for Pro while the view has no plan).
func seedBusinesses(ctx context.Context, t *testing.T, pool *pgxpool.Pool, n int) {
	t.Helper()

	steps := []string{
		`SELECT setseed(0.42)`,
		fmt.Sprintf(`INSERT INTO users (email, password_hash, role)
		 SELECT 'seed' || g || '@example.com', 'hash', 'business_admin' FROM generate_series(1, %d) g`, n),
		`INSERT INTO businesses (owner_user_id, category_id, name, slug, whatsapp_number, status, pilot, verified_at)
		 SELECT u.id,
		        (SELECT array_agg(id ORDER BY id) FROM categories)[1 + (u.id % 7)::int],
		        'Seed ' || u.id, 'seed-' || u.id, '5491112345678',
		        CASE WHEN u.id % 10 = 0 THEN 'paused' ELSE 'active' END,
		        u.id % 20 = 0,
		        CASE WHEN u.id % 3 = 0 THEN now() END
		 FROM users u WHERE u.email LIKE 'seed%'`,
		`INSERT INTO subscriptions (business_id, plan, paid_until, confirmed_by)
		 SELECT b.id, 'starter', CURRENT_DATE + 30, b.owner_user_id FROM businesses b WHERE b.slug LIKE 'seed-%'`,
		`INSERT INTO branches (business_id, name, address, city, location, active)
		 SELECT b.id, 'Sede', 'Calle ' || b.id, 'Ciudad',
		        CASE WHEN b.id % 10 < 6
		             THEN ST_SetSRID(ST_MakePoint(-64.11 + (random() - 0.5) * 0.6, -32.17 + (random() - 0.5) * 0.6), 4326)::geography
		             ELSE ST_SetSRID(ST_MakePoint(-73 + random() * 19, -55 + random() * 33), 4326)::geography
		        END,
		        b.id % 20 <> 1
		 FROM businesses b WHERE b.slug LIKE 'seed-%'`,
		`ANALYZE`,
	}
	for _, sql := range steps {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatalf("seed: %v\n%s", err, sql)
		}
	}
}

func p95(d []time.Duration) time.Duration {
	slices.Sort(d)
	return d[len(d)*95/100]
}

// measure runs fn samples times after a warm-up and returns the p50, p95 and max.
func measure(t *testing.T, fn func(i int) error) (p50, p95v, worst time.Duration) {
	t.Helper()

	for i := range 20 {
		if err := fn(i); err != nil {
			t.Fatalf("warm-up: %v", err)
		}
	}
	d := make([]time.Duration, 0, samples)
	for i := range samples {
		start := time.Now()
		if err := fn(i); err != nil {
			t.Fatalf("query: %v", err)
		}
		d = append(d, time.Since(start))
	}
	p := p95(d)
	return d[len(d)/2], p, d[len(d)-1]
}

func explain(ctx context.Context, t *testing.T, pool *pgxpool.Pool, sql string, args ...any) string {
	t.Helper()

	rows, err := pool.Query(ctx, "EXPLAIN (ANALYZE, BUFFERS) "+sql, args...)
	if err != nil {
		t.Fatalf("explain: %v", err)
	}
	defer rows.Close()

	var plan strings.Builder
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatalf("scan plan: %v", err)
		}
		plan.WriteString(line + "\n")
	}
	return plan.String()
}

// newSeededPool returns a migrated database holding the seeded businesses, with the
// view redefined so the pilot flag stands in for the Pro plan.
func newSeededPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := t.Context()
	pool := dbtest.New(t)
	seedBusinesses(ctx, t, pool, seededBusinesses)
	if _, err := pool.Exec(ctx, `CREATE OR REPLACE VIEW visible_branches AS
		SELECT br.id AS branch_id, b.id AS business_id, b.category_id, b.name AS business_name,
		       b.slug AS business_slug, br.name AS branch_name, br.address, br.city, br.reference,
		       br.location, br.rating_avg, br.rating_count, b.verified_at, b.pilot AS featured
		FROM branches br JOIN businesses b ON b.id = br.business_id
		WHERE br.active AND b.status = 'active'`); err != nil {
		t.Fatalf("redefine view: %v", err)
	}
	return pool
}

// TestSearchLatencyOverTenThousandBusinesses seeds 10k businesses, prints the
// EXPLAIN ANALYZE of both queries and fails when p95 goes over the budget. It only
// runs with -tags perf: it is a measurement, not a regression test, and its timing
// depends on the machine.
//
//	go test -tags perf -run SearchLatency -v ./internal/search/adapters/postgres
func TestSearchLatencyOverTenThousandBusinesses(t *testing.T) {
	ctx := t.Context()
	pool := newSeededPool(t)

	var visible int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM visible_branches`).Scan(&visible); err != nil {
		t.Fatalf("count: %v", err)
	}
	t.Logf("seeded %d businesses, %d visible branches", seededBusinesses, visible)

	repo := NewBranches(pool)
	rng := rand.New(rand.NewPCG(1, 2))
	metro := func() (lat, lng float64) {
		return -32.17 + (rng.Float64()-0.5)*0.4, -64.11 + (rng.Float64()-0.5)*0.4
	}

	type scenario struct {
		name string
		fn   func(i int) error
	}
	scenarios := []scenario{
		{"nearby 5 km, metro", func(int) error {
			lat, lng := metro()
			_, err := repo.Nearby(ctx, search.Query{Lat: lat, Lng: lng, RadiusM: 5_000, Limit: 21})
			return err
		}},
		{"nearby 50 km, metro", func(int) error {
			lat, lng := metro()
			_, err := repo.Nearby(ctx, search.Query{Lat: lat, Lng: lng, RadiusM: search.MaxRadiusM, Limit: 21})
			return err
		}},
		{"nearby 50 km, dense centre", func(int) error {
			// The centre of the dense area is where PostGIS underestimated the most.
			_, err := repo.Nearby(ctx, search.Query{Lat: -32.17, Lng: -64.11, RadiusM: search.MaxRadiusM, Limit: 21})
			return err
		}},
		{"nearby 5 km + category", func(int) error {
			lat, lng := metro()
			_, err := repo.Nearby(ctx, search.Query{Lat: lat, Lng: lng, RadiusM: 5_000, Category: "yoga", Limit: 21})
			return err
		}},
		{"nearby 5 km, sparse area", func(int) error {
			_, err := repo.Nearby(ctx, search.Query{Lat: -38 + rng.Float64(), Lng: -68 + rng.Float64(), RadiusM: 5_000, Limit: 21})
			return err
		}},
		{"featured 5 km, metro", func(i int) error {
			lat, lng := metro()
			_, err := repo.Featured(ctx, search.Query{Lat: lat, Lng: lng, RadiusM: 5_000, Limit: 6}, int64(i))
			return err
		}},
	}

	for _, s := range scenarios {
		p50, p95v, worst := measure(t, s.fn)
		t.Logf("%-28s p50 %-10v p95 %-10v max %v", s.name, p50, p95v, worst)
		if p95v >= budget {
			t.Errorf("%s: p95 %v is over the %v budget", s.name, p95v, budget)
		}
	}

	t.Logf("EXPLAIN nearby 5 km:\n%s", explain(ctx, t, pool, nearbySQL,
		-32.17, -64.11, 5_000.0, float64(search.DistanceBandM), "", 21, 0))
	t.Logf("EXPLAIN nearby 50 km:\n%s", explain(ctx, t, pool, nearbySQL,
		-32.17, -64.11, float64(search.MaxRadiusM), float64(search.DistanceBandM), "", 21, 0))
	t.Logf("EXPLAIN featured 5 km:\n%s", explain(ctx, t, pool, featuredSQL,
		-32.17, -64.11, 5_000.0, int64(1), 6))
}
