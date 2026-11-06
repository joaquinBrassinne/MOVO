package queries_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/search"
	"github.com/joaquinBrassinne/MOVO/internal/search/app/queries"
)

type fakeFeatured struct {
	got   search.Query
	seed  int64
	calls int
}

func (f *fakeFeatured) Featured(_ context.Context, q search.Query, seed int64) ([]search.Branch, error) {
	f.calls++
	f.got, f.seed = q, seed
	return nil, nil
}

func clockAt(t time.Time) func() time.Time { return func() time.Time { return t } }

func TestFeaturedBranchesAppliesDefaults(t *testing.T) {
	t.Parallel()

	finder := &fakeFeatured{}
	if _, err := queries.NewFeaturedBranches(finder, nil).Handle(t.Context(), search.Query{Lat: -32.17, Lng: -64.11}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if finder.got.RadiusM != queries.DefaultRadiusM || finder.got.Limit != queries.DefaultFeaturedLimit {
		t.Errorf("asked radius %d limit %d; want %d and %d",
			finder.got.RadiusM, finder.got.Limit, queries.DefaultRadiusM, queries.DefaultFeaturedLimit)
	}
}

func TestFeaturedBranchesSeedHoldsWithinAWindowAndMovesAfterIt(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	seedAt := func(at time.Time) int64 {
		finder := &fakeFeatured{}
		if _, err := queries.NewFeaturedBranches(finder, clockAt(at)).Handle(t.Context(), search.Query{}); err != nil {
			t.Fatalf("Handle: %v", err)
		}
		return finder.seed
	}

	first := seedAt(start)
	if got := seedAt(start.Add(queries.RotationWindow - time.Second)); got != first {
		t.Errorf("seed moved inside the window: %d then %d", first, got)
	}
	if got := seedAt(start.Add(queries.RotationWindow)); got == first {
		t.Errorf("seed did not move after the window: %d", got)
	}
}

func TestFeaturedBranchesRejectsAnOutOfRangeQueryBeforeReading(t *testing.T) {
	t.Parallel()

	for field, q := range map[string]search.Query{
		"lat":      {Lat: 91},
		"radius_m": {RadiusM: search.MaxRadiusM + 1},
		"limit":    {Limit: queries.MaxFeaturedLimit + 1},
	} {
		finder := &fakeFeatured{}
		_, err := queries.NewFeaturedBranches(finder, nil).Handle(t.Context(), q)

		var qe *search.QueryError
		if !errors.As(err, &qe) || qe.Field != field {
			t.Errorf("%s: err = %v, want a query error on that field", field, err)
		}
		if finder.calls != 0 {
			t.Errorf("%s: the finder was called", field)
		}
	}
}
