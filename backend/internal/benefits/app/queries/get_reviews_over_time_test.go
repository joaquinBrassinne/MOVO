package queries_test

import (
	"context"
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/queries"
)

type fakeReviewsOverTime struct {
	got   benefits.ReviewsOverTime
	asked int
}

func (f *fakeReviewsOverTime) ReviewsOverTime(context.Context, int64, benefits.MetricsPeriod, benefits.MetricsPeriod) (benefits.ReviewsOverTime, error) {
	f.asked++
	return f.got, nil
}

func avg(v float64) *float64 { return &v }

func TestReviewsOverTimeComparesTheAverageAndGivesTheShareThatReviews(t *testing.T) {
	t.Parallel()

	reader := &fakeReviewsOverTime{got: benefits.ReviewsOverTime{
		Current:  benefits.ReviewStats{Reviews: 8, Average: avg(4.5), Redeemed: 20, WithReview: 8, CouponsGenerated: 30},
		Previous: benefits.ReviewStats{Reviews: 5, Average: avg(4.0), Redeemed: 20, WithReview: 5, CouponsGenerated: 25},
	}}
	got, err := queries.NewGetReviewsOverTime(proGate(), reader, clock).Handle(t.Context(), 7, 30)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != queries.StatusOK || got.Rating.Reviews != 8 || got.Rating.PreviousReviews != 5 {
		t.Fatalf("result = %+v, want ok with 8 reviews against 5", got)
	}
	near(t, "average", got.Rating.Average, 4.5)
	near(t, "previous average", got.Rating.Previous, 4)
	near(t, "change in stars", got.Rating.Change, 0.5)
	near(t, "review rate", got.ReviewRate.Value, 0.4)
	near(t, "previous review rate", got.ReviewRate.Previous, 0.25)
	near(t, "review rate change", got.ReviewRate.Change, 0.15)
}

func TestReviewsOverTimeRoundsTheAverageToTwoDecimals(t *testing.T) {
	t.Parallel()

	reader := &fakeReviewsOverTime{got: benefits.ReviewsOverTime{
		Current:  benefits.ReviewStats{Reviews: 3, Average: avg(4.333333333), CouponsGenerated: 25},
		Previous: benefits.ReviewStats{Reviews: 3, Average: avg(4.666666666), CouponsGenerated: 25},
	}}
	got, _ := queries.NewGetReviewsOverTime(proGate(), reader, clock).Handle(t.Context(), 7, 30)
	near(t, "average", got.Rating.Average, 4.33)
	near(t, "previous", got.Rating.Previous, 4.67)
	near(t, "change", got.Rating.Change, -0.34)
}

func TestReviewsOverTimeWithNoReviewsOrNoBaselineHasNothingToCompare(t *testing.T) {
	t.Parallel()

	reader := &fakeReviewsOverTime{got: benefits.ReviewsOverTime{
		Current:  benefits.ReviewStats{Redeemed: 10, CouponsGenerated: 25},
		Previous: benefits.ReviewStats{Reviews: 4, Average: avg(4.5), Redeemed: 10, WithReview: 4, CouponsGenerated: 25},
	}}
	got, err := queries.NewGetReviewsOverTime(proGate(), reader, clock).Handle(t.Context(), 7, 30)
	if err != nil || got.Rating.Average != nil || got.Rating.Change != nil || got.Rating.Reviews != 0 {
		t.Errorf("no reviews = %+v, %v; want no average and no change", got.Rating, err)
	}
	near(t, "review rate", got.ReviewRate.Value, 0)

	// A previous period of 19 coupons is not a baseline, reviews or not.
	reader.got = benefits.ReviewsOverTime{
		Current:  benefits.ReviewStats{Reviews: 3, Average: avg(4), Redeemed: 10, WithReview: 3, CouponsGenerated: 25},
		Previous: benefits.ReviewStats{Reviews: 4, Average: avg(5), Redeemed: 10, WithReview: 4, CouponsGenerated: 19},
	}
	got, _ = queries.NewGetReviewsOverTime(proGate(), reader, clock).Handle(t.Context(), 7, 30)
	if got.Rating.Previous != nil || got.Rating.Change != nil || got.ReviewRate.Previous != nil {
		t.Errorf("baseline from 19 coupons: %+v, %+v", got.Rating, got.ReviewRate)
	}
	if got.Rating.PreviousReviews != 4 {
		t.Errorf("previous reviews = %d, want the count kept", got.Rating.PreviousReviews)
	}
}

func TestReviewsOverTimeNeedsTwentyCouponsAndTheProPlan(t *testing.T) {
	t.Parallel()

	reader := &fakeReviewsOverTime{got: benefits.ReviewsOverTime{Current: benefits.ReviewStats{Reviews: 5, Average: avg(5), CouponsGenerated: 19}}}
	got, err := queries.NewGetReviewsOverTime(proGate(), reader, clock).Handle(t.Context(), 7, 30)
	if err != nil || got.Status != queries.StatusInsufficientData || got.Rating.Average != nil || got.CouponsGenerated != 19 {
		t.Errorf("19 coupons = %+v, %v; want insufficient_data and no average", got, err)
	}

	reader.asked = 0
	denied := queries.NewProGate(fakeOwners{business: 42}, &fakePermission{allowed: false})
	if _, err := queries.NewGetReviewsOverTime(denied, reader, clock).Handle(t.Context(), 7, 30); !errors.Is(err, benefits.ErrPlanUpgradeRequired) || reader.asked != 0 {
		t.Errorf("Starter: err = %v, reads = %d; want ErrPlanUpgradeRequired and no read", err, reader.asked)
	}
	if _, err := queries.NewGetReviewsOverTime(proGate(), reader, clock).Handle(t.Context(), 7, 14); !errors.Is(err, benefits.ErrMetricsPeriodInvalid) || reader.asked != 0 {
		t.Errorf("period 14: err = %v, reads = %d; want ErrMetricsPeriodInvalid and no read", err, reader.asked)
	}
}
