// Package queries holds the read use cases of the search module.
package queries

import (
	"context"

	"github.com/joaquinBrassinne/MOVO/internal/search"
)

// Defaults for the optional parts of a search.
const (
	DefaultRadiusM = 5_000
	DefaultLimit   = 20
)

// Page is one page of results; HasMore tells whether another page follows.
type Page struct {
	Items   []search.Branch
	HasMore bool
}

// NearbyBranches serves the public search. A zero RadiusM or Limit takes its default.
type NearbyBranches struct {
	finder search.BranchFinder
}

// NewNearbyBranches builds the query over finder.
func NewNearbyBranches(finder search.BranchFinder) *NearbyBranches {
	return &NearbyBranches{finder: finder}
}

// Handle validates q, then reads one page. It asks the finder for one extra row to
// know whether more follow, and drops it from the page.
func (n *NearbyBranches) Handle(ctx context.Context, q search.Query) (Page, error) {
	if q.RadiusM == 0 {
		q.RadiusM = DefaultRadiusM
	}
	if q.Limit == 0 {
		q.Limit = DefaultLimit
	}
	if err := q.Validate(); err != nil {
		return Page{}, err
	}

	asked := q
	asked.Limit++
	items, err := n.finder.Nearby(ctx, asked)
	if err != nil {
		return Page{}, err
	}
	if len(items) > q.Limit {
		return Page{Items: items[:q.Limit], HasMore: true}, nil
	}
	return Page{Items: items}, nil
}
