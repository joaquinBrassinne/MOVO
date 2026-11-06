package queries

import (
	"context"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/reviews"
)

// GetRatingSummary serves the header of "Reseñas del local": the average and how many
// reviews gave each number of stars (SDD 5.5).
type GetRatingSummary struct {
	reader reviews.ReviewReader
}

// NewGetRatingSummary wires the GetRatingSummary query.
func NewGetRatingSummary(reader reviews.ReviewReader) *GetRatingSummary {
	return &GetRatingSummary{reader: reader}
}

// Handle returns the summary, or reviews.ErrBranchNotFound for a branch the public cannot see.
func (q *GetRatingSummary) Handle(ctx context.Context, branchID int64) (reviews.RatingSummary, error) {
	got, err := q.reader.Summary(ctx, branchID)
	if err != nil {
		return reviews.RatingSummary{}, fmt.Errorf("rating summary: %w", err)
	}
	return got, nil
}
