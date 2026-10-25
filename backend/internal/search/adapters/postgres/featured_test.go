package postgres_test

import (
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
	"github.com/joaquinBrassinne/MOVO/internal/search"
	"github.com/joaquinBrassinne/MOVO/internal/search/adapters/postgres"
)

// featuredAsPilot redefines the view so the pilot flag stands in for the Pro plan,
// which does not exist yet. The columns are the ones of the real view.
func featuredAsPilot(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(t.Context(), `CREATE OR REPLACE VIEW visible_branches AS
		SELECT br.id AS branch_id, b.id AS business_id, b.category_id, b.name AS business_name,
		       b.slug AS business_slug, br.name AS branch_name, br.address, br.city, br.reference,
		       br.location, br.rating_avg, br.rating_count, b.verified_at, b.pilot AS featured
		FROM branches br JOIN businesses b ON b.id = br.business_id
		WHERE br.active AND b.status = 'active'`); err != nil {
		t.Fatalf("redefine view: %v", err)
	}
}

func featured(t *testing.T, repo *postgres.Branches, q search.Query, seed int64) []search.Branch {
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
	found, err := repo.Featured(t.Context(), q, seed)
	if err != nil {
		t.Fatalf("Featured: %v", err)
	}
	return found
}

func ids(branches []search.Branch) []int64 {
	out := make([]int64, 0, len(branches))
	for _, b := range branches {
		out = append(out, b.BranchID)
	}
	return out
}

func TestFeaturedIsEmptyWhileNoBusinessIsFeatured(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	s := &seeded{pool: pool}
	s.add(t, "yoga", "active", 500, true) // pilot, but the real view does not feature it yet

	if got := featured(t, postgres.NewBranches(pool), search.Query{}, 1); len(got) != 0 {
		t.Errorf("featured = %v, want none", ids(got))
	}
}

func TestFeaturedKeepsFeaturedVisibleBusinessesInsideTheRadius(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	featuredAsPilot(t, pool)
	s := &seeded{pool: pool}
	want := s.add(t, "yoga", "active", 1_000, true)
	s.add(t, "yoga", "active", 500, false)  // not featured
	s.add(t, "yoga", "active", 9_000, true) // outside a 5 km radius
	s.add(t, "yoga", "paused", 500, true)   // not visible

	got := featured(t, postgres.NewBranches(pool), search.Query{RadiusM: 5_000}, 1)
	if !slices.Equal(ids(got), []int64{want}) || !got[0].Featured {
		t.Errorf("featured = %v, want only branch %d marked featured", ids(got), want)
	}
}

func TestFeaturedListsEachBusinessOnceThroughItsNearestBranch(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	featuredAsPilot(t, pool)
	s := &seeded{pool: pool}
	far := s.add(t, "yoga", "active", 3_000, true)
	var business int64
	if err := pool.QueryRow(t.Context(), `SELECT business_id FROM branches WHERE id = $1`, far).Scan(&business); err != nil {
		t.Fatalf("business of branch: %v", err)
	}
	var near int64
	if err := pool.QueryRow(t.Context(),
		`INSERT INTO branches (business_id, name, address, city, location)
		 VALUES ($1, 'Norte', 'Calle 2', 'Rio Tercero', ST_SetSRID(ST_MakePoint($2::float8, $3::float8), 4326)::geography) RETURNING id`,
		business, originLng, originLat+500/metersPerDegree).Scan(&near); err != nil {
		t.Fatalf("insert second branch: %v", err)
	}

	got := featured(t, postgres.NewBranches(pool), search.Query{}, 1)
	if !slices.Equal(ids(got), []int64{near}) {
		t.Errorf("featured = %v, want the nearest branch %d once", ids(got), near)
	}
}

// TestFeaturedRotatesWithTheSeed pins that one seed gives one order and that other
// seeds reorder the businesses, so each one leads sometimes.
func TestFeaturedRotatesWithTheSeed(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	featuredAsPilot(t, pool)
	s := &seeded{pool: pool}
	for i := 1; i <= 4; i++ {
		s.add(t, "yoga", "active", float64(i)*100, true)
	}
	repo := postgres.NewBranches(pool)

	if a, b := ids(featured(t, repo, search.Query{}, 7)), ids(featured(t, repo, search.Query{}, 7)); !slices.Equal(a, b) {
		t.Errorf("the same seed gave %v and %v", a, b)
	}
	leaders := map[int64]bool{}
	for seed := int64(0); seed < 40; seed++ {
		got := featured(t, repo, search.Query{}, seed)
		if len(got) != 4 {
			t.Fatalf("seed %d: %d businesses, want all 4", seed, len(got))
		}
		leaders[got[0].BranchID] = true
	}
	if len(leaders) < 2 {
		t.Errorf("only %d business ever led in 40 seeds; want the lead to rotate", len(leaders))
	}
}

func TestFeaturedHonoursTheLimit(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	featuredAsPilot(t, pool)
	s := &seeded{pool: pool}
	for i := 1; i <= 3; i++ {
		s.add(t, "yoga", "active", float64(i)*100, true)
	}

	if got := featured(t, postgres.NewBranches(pool), search.Query{Limit: 2}, 1); len(got) != 2 {
		t.Errorf("got %d, want 2", len(got))
	}
}
