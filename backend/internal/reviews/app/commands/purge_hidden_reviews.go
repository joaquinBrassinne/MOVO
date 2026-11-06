package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/reviews"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

// PurgeHiddenReviews is the daily sweep that erases the text of reviews hidden for longer
// than domain.HiddenRetention (SDD 5.5).
type PurgeHiddenReviews struct {
	purger reviews.HiddenReviewPurger
	now    func() time.Time
}

// NewPurgeHiddenReviews wires the PurgeHiddenReviews use case. A nil clock uses time.Now.
func NewPurgeHiddenReviews(purger reviews.HiddenReviewPurger, now func() time.Time) *PurgeHiddenReviews {
	if now == nil {
		now = time.Now
	}
	return &PurgeHiddenReviews{purger: purger, now: now}
}

// Handle purges the reviews hidden before now minus the retention and returns how many.
func (c *PurgeHiddenReviews) Handle(ctx context.Context) (int64, error) {
	n, err := c.purger.PurgeHidden(ctx, c.now().Add(-domain.HiddenRetention))
	if err != nil {
		return 0, fmt.Errorf("purge hidden reviews: %w", err)
	}
	return n, nil
}
