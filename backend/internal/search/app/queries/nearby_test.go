package queries_test

import (
	"context"
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/search"
	"github.com/joaquinBrassinne/MOVO/internal/search/app/queries"
)

// fakeFinder returns n branches and records the query it was asked.
type fakeFinder struct {
	n    int
	got  search.Query
	err  error
	call int
}

func (f *fakeFinder) Nearby(_ context.Context, q search.Query) ([]search.Branch, error) {
	f.call++
	f.got = q
	if f.err != nil {
		return nil, f.err
	}
	out := make([]search.Branch, min(f.n, q.Limit))
	return out, nil
}

func TestNearbyBranchesAppliesDefaults(t *testing.T) {
	t.Parallel()

	finder := &fakeFinder{}
	if _, err := queries.NewNearbyBranches(finder).Handle(t.Context(), search.Query{Lat: -32.17, Lng: -64.11}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if finder.got.RadiusM != queries.DefaultRadiusM {
		t.Errorf("radius = %d, want %d", finder.got.RadiusM, queries.DefaultRadiusM)
	}
	if want := queries.DefaultLimit + 1; finder.got.Limit != want {
		t.Errorf("asked limit = %d, want %d (page plus one)", finder.got.Limit, want)
	}
}

func TestNearbyBranchesReportsWhetherAnotherPageFollows(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		available int
		wantItems int
		wantMore  bool
	}{
		{available: 5, wantItems: 5, wantMore: false},
		{available: 10, wantItems: 10, wantMore: false},
		{available: 11, wantItems: 10, wantMore: true},
	} {
		page, err := queries.NewNearbyBranches(&fakeFinder{n: tc.available}).Handle(t.Context(), search.Query{Limit: 10})
		if err != nil {
			t.Fatalf("Handle: %v", err)
		}
		if len(page.Items) != tc.wantItems || page.HasMore != tc.wantMore {
			t.Errorf("%d available: %d items, more=%t; want %d, %t",
				tc.available, len(page.Items), page.HasMore, tc.wantItems, tc.wantMore)
		}
	}
}

func TestNearbyBranchesRejectsInvalidQueriesBeforeReading(t *testing.T) {
	t.Parallel()

	valid := search.Query{Lat: -32.17, Lng: -64.11, RadiusM: 1_000, Limit: 10}
	for field, mutate := range map[string]func(*search.Query){
		"lat":      func(q *search.Query) { q.Lat = 91 },
		"lng":      func(q *search.Query) { q.Lng = -181 },
		"radius_m": func(q *search.Query) { q.RadiusM = search.MaxRadiusM + 1 },
		"limit":    func(q *search.Query) { q.Limit = search.MaxLimit + 1 },
		"offset":   func(q *search.Query) { q.Offset = -1 },
	} {
		finder := &fakeFinder{}
		q := valid
		mutate(&q)

		_, err := queries.NewNearbyBranches(finder).Handle(t.Context(), q)

		var qe *search.QueryError
		if !errors.Is(err, search.ErrInvalidQuery) || !errors.As(err, &qe) || qe.Field != field {
			t.Errorf("%s: err = %v, want a query error on that field", field, err)
		}
		if finder.call != 0 {
			t.Errorf("%s: the finder was called for an invalid query", field)
		}
	}
}
