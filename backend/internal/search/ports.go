// Package search finds the public branches near a point. It reads only from the
// visible_branches view, so what is public is decided in one place (SDD 4.1).
package search

import (
	"context"
	"errors"
)

// ErrInvalidQuery is returned when the point, radius or paging of a query is out of range.
var ErrInvalidQuery = errors.New("search: invalid query")

// QueryError names the query field that is out of range. It matches ErrInvalidQuery.
type QueryError struct{ Field string }

func (e *QueryError) Error() string { return "search: invalid query: " + e.Field }

func (e *QueryError) Is(target error) bool { return target == ErrInvalidQuery }

const (
	// MaxRadiusM is the widest radius a client may ask for.
	MaxRadiusM = 50_000
	// MaxLimit is the largest page a client may ask for.
	MaxLimit = 50
	// DistanceBandM is the width of a distance band: within a band, featured branches
	// come first, so a far Pro branch never beats a much closer one (SDD 4.2).
	DistanceBandM = 2_000
	// MinReviewsToRate is how many reviews a branch needs before its rating is shown;
	// with fewer it reads "Nuevo" (SDD 5.5). The minimum-rating filter applies the same
	// threshold, so it never keeps a branch whose card shows no stars.
	MinReviewsToRate = 3
)

// Query is a proximity search. Category is a category slug; empty means all. MinRating
// is an optional filter of 0 to 5 stars, 0 meaning none; it filters and never orders
// (SDD 5.5).
type Query struct {
	Lat, Lng  float64
	RadiusM   int
	Category  string
	MinRating float64
	Limit     int
	Offset    int
}

// Branch is a public branch with its distance to the searched point.
type Branch struct {
	BranchID     int64
	BusinessID   int64
	BusinessName string
	BusinessSlug string
	BranchName   string
	Address      string
	City         string
	Reference    string
	Lat, Lng     float64
	DistanceM    float64
	RatingAvg    float64
	RatingCount  int
	Verified     bool
	Featured     bool
	CategorySlug string
	CategoryName string
	// BenefitLabel is the label of the best benefit the business offers (a percent over
	// an amount over a custom one), empty when it offers none.
	BenefitLabel string
}

// BranchFinder lists the visible branches around a point, ordered by distance band,
// then featured first, then exact distance.
type BranchFinder interface {
	Nearby(ctx context.Context, q Query) ([]Branch, error)
}

// Validate reports a *QueryError when q cannot be searched.
func (q Query) Validate() error {
	switch {
	case q.Lat < -90 || q.Lat > 90:
		return &QueryError{"lat"}
	case q.Lng < -180 || q.Lng > 180:
		return &QueryError{"lng"}
	case q.RadiusM <= 0 || q.RadiusM > MaxRadiusM:
		return &QueryError{"radius_m"}
	case !(q.MinRating >= 0 && q.MinRating <= 5): // written so NaN is refused too
		return &QueryError{"min_rating"}
	case q.Limit <= 0 || q.Limit > MaxLimit:
		return &QueryError{"limit"}
	case q.Offset < 0:
		return &QueryError{"offset"}
	}
	return nil
}

// FeaturedFinder lists the featured (Pro) businesses around a point, one branch per
// business, in an order that depends on seed: the same seed gives the same order and
// a new seed reshuffles it.
type FeaturedFinder interface {
	Featured(ctx context.Context, q Query, seed int64) ([]Branch, error)
}
