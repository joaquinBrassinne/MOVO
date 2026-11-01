package queries

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
)

// RatingMovement is the average rating of the reviews written in a period against the average of
// the period before. Average is nil with no reviews; Previous and Change are nil without a
// baseline (the previous period had fewer than benefits.MinimumSample coupons or no reviews).
// Change is Average minus Previous, in stars.
type RatingMovement struct {
	Reviews         int64
	PreviousReviews int64
	Average         *float64
	Previous        *float64
	Change          *float64
}

// ReviewsOverTimeResult is how the reviews of a business moved. ReviewRate is the share of the
// coupons redeemed in the period that have a review. With status insufficient_data only
// CouponsGenerated and Minimum are meaningful.
type ReviewsOverTimeResult struct {
	Status           string
	Period           benefits.MetricsPeriod
	PreviousPeriod   benefits.MetricsPeriod
	CouponsGenerated int64
	Minimum          int

	Rating     RatingMovement
	ReviewRate Rate
}

// GetReviewsOverTime answers how the reviews evolve (SDD 6.16), a Pro metric: the average rating
// of the period against the previous one, and the share of redemptions that leave a review.
type GetReviewsOverTime struct {
	gate   ProGate
	reader benefits.ReviewsOverTimeReader
	now    func() time.Time
}

// NewGetReviewsOverTime wires the query. A nil clock uses time.Now.
func NewGetReviewsOverTime(gate ProGate, reader benefits.ReviewsOverTimeReader, now func() time.Time) *GetReviewsOverTime {
	if now == nil {
		now = time.Now
	}
	return &GetReviewsOverTime{gate: gate, reader: reader, now: now}
}

// Handle checks the plan first, then reads the period asked for (7, 30 or 90 days, 30 by default).
func (q *GetReviewsOverTime) Handle(ctx context.Context, ownerID int64, days int) (ReviewsOverTimeResult, error) {
	businessID, err := q.gate.Business(ctx, ownerID)
	if err != nil {
		return ReviewsOverTimeResult{}, err
	}
	days, err = validDays(days)
	if err != nil {
		return ReviewsOverTimeResult{}, err
	}

	current := periodOf(q.now(), days)
	previous := previousPeriod(current)
	got, err := q.reader.ReviewsOverTime(ctx, businessID, current, previous)
	if err != nil {
		return ReviewsOverTimeResult{}, fmt.Errorf("read reviews over time: %w", err)
	}

	result := ReviewsOverTimeResult{
		Status: StatusOK, Period: current, PreviousPeriod: previous,
		CouponsGenerated: got.Current.CouponsGenerated, Minimum: benefits.MinimumSample,
	}
	if got.Current.CouponsGenerated < benefits.MinimumSample {
		result.Status = StatusInsufficientData
		return result, nil
	}

	comparable := got.Previous.CouponsGenerated >= benefits.MinimumSample
	result.Rating = RatingMovement{Reviews: got.Current.Reviews, PreviousReviews: got.Previous.Reviews, Average: stars(got.Current.Average)}
	if comparable {
		result.Rating.Previous = stars(got.Previous.Average)
	}
	if result.Rating.Average != nil && result.Rating.Previous != nil {
		change := math.Round((*result.Rating.Average-*result.Rating.Previous)*100) / 100
		result.Rating.Change = &change
	}
	result.ReviewRate = rate(got.Current.WithReview, got.Current.Redeemed, got.Previous.WithReview, got.Previous.Redeemed, comparable)
	return result, nil
}

// stars rounds an average to two decimals, keeping nil.
func stars(v *float64) *float64 {
	if v == nil {
		return nil
	}
	r := math.Round(*v*100) / 100
	return &r
}
