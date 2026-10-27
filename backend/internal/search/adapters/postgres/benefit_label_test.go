package postgres_test

import (
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
	"github.com/joaquinBrassinne/MOVO/internal/search"
	"github.com/joaquinBrassinne/MOVO/internal/search/adapters/postgres"
)

// offer inserts a benefit of the business of the branch. admin is the super admin's pause
// and off the owner's switch turned off.
func offer(t *testing.T, s *seeded, branch int64, kind string, percent, amount any, admin, off bool) {
	t.Helper()
	title := any(nil)
	if kind == "custom" {
		title = "Regalo"
	}
	if _, err := s.pool.Exec(t.Context(), `
INSERT INTO benefits (business_id, type, percent, amount_ars, title, active, admin_paused)
VALUES ($1, $2, $3, $4, $5, NOT $6, $7)`,
		businessOf(t, s.pool, branch), kind, percent, amount, title, off, admin); err != nil {
		t.Fatalf("insert benefit: %v", err)
	}
}

func labelsByBranch(branches []search.Branch) map[int64]string {
	out := map[int64]string{}
	for _, b := range branches {
		out[b.BranchID] = b.BenefitLabel
	}
	return out
}

func TestCardsCarryTheLabelOfTheBestBenefitTheBusinessOffers(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	s := &seeded{pool: pool}
	repo := postgres.NewBranches(pool)

	percentOverAmount := s.add(t, "yoga", "active", 100, false)
	offer(t, s, percentOverAmount, "percent", 10, nil, false, false)
	offer(t, s, percentOverAmount, "percent", 20, nil, false, false)
	offer(t, s, percentOverAmount, "amount", nil, 5000, false, false)

	biggestAmount := s.add(t, "yoga", "active", 200, false)
	offer(t, s, biggestAmount, "amount", nil, 2000, false, false)
	offer(t, s, biggestAmount, "amount", nil, 3000, false, false)
	offer(t, s, biggestAmount, "custom", nil, nil, false, false)

	onlyCustom := s.add(t, "yoga", "active", 300, false)
	offer(t, s, onlyCustom, "custom", nil, nil, false, false)

	nothing := s.add(t, "yoga", "active", 400, false)

	// A paused benefit and one the owner turned off do not count, however big they are.
	hidden := s.add(t, "yoga", "active", 500, false)
	offer(t, s, hidden, "percent", 50, nil, true, false)
	offer(t, s, hidden, "percent", 30, nil, false, true)
	offer(t, s, hidden, "amount", nil, 500, false, false)

	found, err := repo.Nearby(t.Context(), search.Query{Lat: originLat, Lng: originLng, RadiusM: 20_000, Limit: 50})
	if err != nil {
		t.Fatalf("Nearby: %v", err)
	}
	got := labelsByBranch(found)
	for branch, want := range map[int64]string{
		percentOverAmount: "-20%",
		biggestAmount:     "-$3.000",
		onlyCustom:        "Beneficio",
		nothing:           "",
		hidden:            "-$500",
	} {
		if got[branch] != want {
			t.Errorf("branch %d label = %q, want %q", branch, got[branch], want)
		}
	}
	if len(found) != 5 {
		t.Errorf("found %d branches, want 5: the benefits must not multiply the rows", len(found))
	}
}

func TestFeaturedCardsCarryTheLabelToo(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	featuredAsPilot(t, pool)
	s := &seeded{pool: pool}
	repo := postgres.NewBranches(pool)

	with := s.add(t, "yoga", "active", 100, true)
	offer(t, s, with, "percent", 15, nil, false, false)
	offer(t, s, with, "percent", 25, nil, false, false)
	without := s.add(t, "yoga", "active", 200, true)

	got := labelsByBranch(featured(t, repo, search.Query{}, 1))
	if got[with] != "-25%" || got[without] != "" || len(got) != 2 {
		t.Errorf("featured labels = %v, want -25%% for the first and none for the second, one row each", got)
	}
}
