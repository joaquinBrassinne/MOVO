package postgres_test

import (
	"slices"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
	"github.com/joaquinBrassinne/MOVO/internal/search"
	"github.com/joaquinBrassinne/MOVO/internal/search/adapters/postgres"
)

// rate sets the rating a branch shows, the way the reviews module keeps it.
func rate(t *testing.T, s *seeded, branch int64, avg string, count int) {
	t.Helper()
	if _, err := s.pool.Exec(t.Context(), `UPDATE branches SET rating_avg = $2::numeric, rating_count = $3 WHERE id = $1`, branch, avg, count); err != nil {
		t.Fatalf("rate branch: %v", err)
	}
}

// TestNearbyFiltersByMinimumRating pins SDD 5.5: the rating is an optional filter that
// never orders. A branch with fewer than 3 reviews shows "Nuevo" and no stars, so it
// cannot claim a minimum and stays out whenever one is asked for.
func TestNearbyFiltersByMinimumRating(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	s := &seeded{pool: pool}
	// Ascending distance, so the expected order is the order they were added in.
	top := s.add(t, "yoga", "active", 500, false)      // 4.50 from 5 reviews
	low := s.add(t, "yoga", "active", 1_000, false)    // 3.00 from 10 reviews
	newbie := s.add(t, "yoga", "active", 1_500, false) // 5.00 from 1 review: "Nuevo"
	two := s.add(t, "yoga", "active", 2_000, false)    // 5.00 from 2 reviews: "Nuevo"
	edge := s.add(t, "yoga", "active", 2_500, false)   // 4.00 from exactly 3 reviews
	none := s.add(t, "yoga", "active", 3_000, false)   // no reviews
	rate(t, s, top, "4.50", 5)
	rate(t, s, low, "3.00", 10)
	rate(t, s, newbie, "5.00", 1)
	rate(t, s, two, "5.00", 2)
	rate(t, s, edge, "4.00", 3)
	repo := postgres.NewBranches(pool)

	tests := []struct {
		name string
		min  float64
		want []int64
	}{
		{"no filter lists everyone", 0, []int64{top, low, newbie, two, edge, none}},
		{"1 star leaves out only the ones without a rating to claim", 1, []int64{top, low, edge}},
		{"4 stars", 4, []int64{top, edge}},
		{"the average itself passes", 4.5, []int64{top}},
		{"just above the average does not", 4.51, nil},
		{"5 stars: a perfect 5 from 1 or 2 reviews is still Nuevo", 5, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := nearby(t, repo, search.Query{MinRating: tt.min})
			if !slices.Equal(got, tt.want) && (len(got) != 0 || len(tt.want) != 0) {
				t.Errorf("min_rating %v = %v, want %v, in distance order", tt.min, got, tt.want)
			}
		})
	}
}

func TestNearbyMinimumRatingCombinesWithCategoryAndRadius(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	s := &seeded{pool: pool}
	yoga := s.add(t, "yoga", "active", 500, false)
	padel := s.add(t, "padel", "active", 600, false)
	far := s.add(t, "yoga", "active", 9_000, false)
	for _, b := range []int64{yoga, padel, far} {
		rate(t, s, b, "4.80", 8)
	}

	got := nearby(t, postgres.NewBranches(pool), search.Query{MinRating: 4, Category: "yoga", RadiusM: 5_000})
	if want := []int64{yoga}; !slices.Equal(got, want) {
		t.Errorf("branches = %v, want only %v: well rated, yoga and inside 5 km", got, want)
	}
}
