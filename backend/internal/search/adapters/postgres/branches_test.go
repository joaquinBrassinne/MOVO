package postgres_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
	"github.com/joaquinBrassinne/MOVO/internal/search"
	"github.com/joaquinBrassinne/MOVO/internal/search/adapters/postgres"
)

// The origin is the centre of Río Tercero; one degree of latitude is about 111 km.
const (
	originLat       = -32.17
	originLng       = -64.11
	metersPerDegree = 111_000.0
)

type seeded struct {
	pool *pgxpool.Pool
	n    int
}

// add inserts a business with one branch distM meters north of the origin and
// returns the branch id. pilot doubles as the featured flag in these tests.
func (s *seeded) add(t *testing.T, category, status string, distM float64, pilot bool) int64 {
	t.Helper()
	s.n++

	var owner, business, branch int64
	if err := s.pool.QueryRow(t.Context(),
		`INSERT INTO users (email, password_hash, role) VALUES ($1, 'hash', 'business_admin') RETURNING id`,
		fmt.Sprintf("owner%d@example.com", s.n)).Scan(&owner); err != nil {
		t.Fatalf("insert owner: %v", err)
	}
	if err := s.pool.QueryRow(t.Context(),
		`INSERT INTO businesses (owner_user_id, category_id, name, slug, whatsapp_number, status, pilot)
		 VALUES ($1, (SELECT id FROM categories WHERE slug = $2), $3, $4, '5491112345678', $5, $6) RETURNING id`,
		owner, category, fmt.Sprintf("Studio %d", s.n), fmt.Sprintf("studio-%d", s.n), status, pilot).Scan(&business); err != nil {
		t.Fatalf("insert business: %v", err)
	}
	// The view shows a business only if it is a pilot or paid; a paid one stands in for
	// "not a pilot" here.
	if _, err := s.pool.Exec(t.Context(),
		`INSERT INTO subscriptions (business_id, plan, paid_until, confirmed_by)
		 VALUES ($1, 'starter', CURRENT_DATE + 30, $2)`, business, owner); err != nil {
		t.Fatalf("insert subscription: %v", err)
	}
	if err := s.pool.QueryRow(t.Context(),
		`INSERT INTO branches (business_id, name, address, city, location)
		 VALUES ($1, 'Centro', 'Calle 1', 'Río Tercero', ST_SetSRID(ST_MakePoint($2::float8, $3::float8), 4326)::geography) RETURNING id`,
		business, originLng, originLat+distM/metersPerDegree).Scan(&branch); err != nil {
		t.Fatalf("insert branch: %v", err)
	}
	return branch
}

func nearby(t *testing.T, repo *postgres.Branches, q search.Query) []int64 {
	t.Helper()
	if q.Lat == 0 && q.Lng == 0 {
		q.Lat, q.Lng = originLat, originLng
	}
	if q.RadiusM == 0 {
		q.RadiusM = 20_000
	}
	if q.Limit == 0 {
		q.Limit = 50
	}
	found, err := repo.Nearby(t.Context(), q)
	if err != nil {
		t.Fatalf("Nearby: %v", err)
	}
	ids := make([]int64, 0, len(found))
	for _, b := range found {
		ids = append(ids, b.BranchID)
	}
	return ids
}

func TestNearbyKeepsOnlyVisibleBranchesInsideTheRadius(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	s := &seeded{pool: pool}
	inside := s.add(t, "yoga", "active", 1_000, false)
	s.add(t, "yoga", "active", 9_000, false) // outside a 5 km radius
	s.add(t, "yoga", "paused", 500, false)   // not visible

	got := nearby(t, postgres.NewBranches(pool), search.Query{RadiusM: 5_000})
	if want := []int64{inside}; !slices.Equal(got, want) {
		t.Errorf("branches = %v, want %v", got, want)
	}
}

func TestNearbyFiltersByCategory(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	s := &seeded{pool: pool}
	yoga := s.add(t, "yoga", "active", 1_000, false)
	s.add(t, "padel", "active", 500, false)
	repo := postgres.NewBranches(pool)

	if got, want := nearby(t, repo, search.Query{Category: "yoga"}), []int64{yoga}; !slices.Equal(got, want) {
		t.Errorf("yoga = %v, want %v", got, want)
	}
	if got := nearby(t, repo, search.Query{Category: "no-such-category"}); len(got) != 0 {
		t.Errorf("unknown category = %v, want none", got)
	}
	if got := nearby(t, repo, search.Query{}); len(got) != 2 {
		t.Errorf("no category = %v, want both", got)
	}
}

// TestNearbyOrdersByBandThenFeaturedThenDistance pins the SDD 4.2 order. The view
// has no plan yet, so the test redefines featured as the pilot flag.
func TestNearbyOrdersByBandThenFeaturedThenDistance(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	s := &seeded{pool: pool}
	if _, err := pool.Exec(t.Context(), `CREATE OR REPLACE VIEW visible_branches AS
		SELECT br.id AS branch_id, b.id AS business_id, b.category_id, b.name AS business_name,
		       b.slug AS business_slug, br.name AS branch_name, br.address, br.city, br.reference,
		       br.location, br.rating_avg, br.rating_count, b.verified_at, b.pilot AS featured
		FROM branches br JOIN businesses b ON b.id = br.business_id
		WHERE br.active AND b.status = 'active'`); err != nil {
		t.Fatalf("redefine view: %v", err)
	}

	plainNear := s.add(t, "yoga", "active", 500, false)      // band 0
	featuredBand0 := s.add(t, "yoga", "active", 1_500, true) // band 0, farther but featured
	plainBand1 := s.add(t, "yoga", "active", 2_100, false)   // band 1
	featuredFar := s.add(t, "yoga", "active", 9_000, true)   // band 4: never beats a closer one

	got := nearby(t, postgres.NewBranches(pool), search.Query{})
	want := []int64{featuredBand0, plainNear, plainBand1, featuredFar}
	if !slices.Equal(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}

func TestNearbyPagesAreStable(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	s := &seeded{pool: pool}
	a := s.add(t, "yoga", "active", 100, false)
	b := s.add(t, "yoga", "active", 200, false)
	c := s.add(t, "yoga", "active", 300, false)
	repo := postgres.NewBranches(pool)

	if got, want := nearby(t, repo, search.Query{Limit: 2}), []int64{a, b}; !slices.Equal(got, want) {
		t.Errorf("page 1 = %v, want %v", got, want)
	}
	if got, want := nearby(t, repo, search.Query{Limit: 2, Offset: 2}), []int64{c}; !slices.Equal(got, want) {
		t.Errorf("page 2 = %v, want %v", got, want)
	}
}
