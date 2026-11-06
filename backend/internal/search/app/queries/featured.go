package queries

import (
	"context"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/search"
)

// Defaults and bounds of the featured section of the Home.
const (
	DefaultFeaturedLimit = 6
	MaxFeaturedLimit     = 20
	// RotationWindow is how long the featured order holds before it reshuffles. A
	// window keeps the Home stable while someone looks at it and still spreads the
	// exposure across the Pro businesses over the day.
	RotationWindow = 15 * time.Minute
)

// FeaturedBranches serves "Destacados cerca tuyo". A zero RadiusM or Limit takes its default.
type FeaturedBranches struct {
	finder search.FeaturedFinder
	now    func() time.Time
}

// NewFeaturedBranches builds the query. A nil clock uses time.Now.
func NewFeaturedBranches(finder search.FeaturedFinder, now func() time.Time) *FeaturedBranches {
	if now == nil {
		now = time.Now
	}
	return &FeaturedBranches{finder: finder, now: now}
}

// Handle returns the featured businesses near q, rotated by the current window.
func (f *FeaturedBranches) Handle(ctx context.Context, q search.Query) ([]search.Branch, error) {
	if q.RadiusM == 0 {
		q.RadiusM = DefaultRadiusM
	}
	if q.Limit == 0 {
		q.Limit = DefaultFeaturedLimit
	}
	if err := q.Validate(); err != nil {
		return nil, err
	}
	if q.Limit > MaxFeaturedLimit {
		return nil, &search.QueryError{Field: "limit"}
	}
	return f.finder.Featured(ctx, q, f.now().Unix()/int64(RotationWindow.Seconds()))
}
