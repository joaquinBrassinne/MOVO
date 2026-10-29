package postgres_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
	"github.com/joaquinBrassinne/MOVO/internal/reviews"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

func TestSummaryCountsThePublicReviewsByStars(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)
	repo := postgres.NewReviews(pool)

	for i, spec := range []struct {
		stars  int
		status string
	}{{5, "visible"}, {5, "reported"}, {4, "visible"}, {1, "visible"}, {5, "hidden"}, {2, "hidden"}} {
		id := addReview(ctx, t, pool, s, i+1, spec.status, created.Add(time.Duration(i)*time.Minute))
		if _, err := pool.Exec(ctx, `UPDATE reviews SET rating = $2 WHERE id = $1`, id, spec.stars); err != nil {
			t.Fatalf("set stars: %v", err)
		}
	}

	got, err := repo.Summary(ctx, s.branch)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	want := reviews.RatingSummary{Count: 4, Stars: [5]int{1, 0, 0, 1, 2}}
	if got != want {
		t.Errorf("summary = %+v, want %+v: a reported review counts and a hidden one does not", got, want)
	}
	if avg := got.Average(); avg != 3.75 {
		t.Errorf("average = %v, want 3.75", avg)
	}

	if empty, err := repo.Summary(ctx, s.branch+1000); !errors.Is(err, reviews.ErrBranchNotFound) {
		t.Errorf("Summary of a branch that does not exist = %+v, %v; want ErrBranchNotFound", empty, err)
	}
}

func TestListPublicFiltersByStars(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)
	repo := postgres.NewReviews(pool)

	byStars := map[int]int64{}
	for stars := 1; stars <= 5; stars++ {
		id := addReview(ctx, t, pool, s, stars, "visible", created.Add(time.Duration(stars)*time.Minute))
		if _, err := pool.Exec(ctx, `UPDATE reviews SET rating = $2 WHERE id = $1`, id, stars); err != nil {
			t.Fatalf("set stars: %v", err)
		}
		byStars[stars] = id
	}

	for name, tt := range map[string]struct {
		filter domain.StarFilter
		want   []int64
	}{
		"no filter":               {domain.StarFilter{}, []int64{byStars[5], byStars[4], byStars[3], byStars[2], byStars[1]}},
		"exactly 5":               {domain.StarFilter{Min: 5, Max: 5}, []int64{byStars[5]}},
		"4 or more":               {domain.StarFilter{Min: 4}, []int64{byStars[5], byStars[4]}},
		"2 or fewer":              {domain.StarFilter{Max: 2}, []int64{byStars[2], byStars[1]}},
		"between 2 and 3":         {domain.StarFilter{Min: 2, Max: 3}, []int64{byStars[3], byStars[2]}},
		"4 or more, cursor after": {domain.StarFilter{Min: 4}, []int64{byStars[4]}},
	} {
		var after *domain.Cursor
		if name == "4 or more, cursor after" {
			after = &domain.Cursor{CreatedAt: created.Add(5 * time.Minute), ID: byStars[5]}
		}
		got, err := repo.ListPublic(ctx, s.branch, tt.filter, after, 10)
		if err != nil || !slices.Equal(ids(got), tt.want) {
			t.Errorf("%s = %v, %v; want %v", name, ids(got), err, tt.want)
		}
	}
}
