package postgres

import (
	"context"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
)

// ReviewsOverTime reads the review stats of the period and of the one before it. It reads the
// reviews table for counts and an average only: no comment, name or customer leaves it.
func (r *Coupons) ReviewsOverTime(ctx context.Context, businessID int64, current, previous benefits.MetricsPeriod) (benefits.ReviewsOverTime, error) {
	var (
		out benefits.ReviewsOverTime
		err error
	)
	if out.Current, err = r.reviewStats(ctx, businessID, current); err != nil {
		return benefits.ReviewsOverTime{}, err
	}
	if out.Previous, err = r.reviewStats(ctx, businessID, previous); err != nil {
		return benefits.ReviewsOverTime{}, err
	}
	return out, nil
}

func (r *Coupons) reviewStats(ctx context.Context, businessID int64, p benefits.MetricsPeriod) (benefits.ReviewStats, error) {
	var out benefits.ReviewStats
	err := r.pool.QueryRow(ctx, `
SELECT
  (SELECT count(*) FROM reviews WHERE business_id = $1 AND status <> 'hidden' AND created_at >= $2 AND created_at < $3),
  (SELECT avg(rating)::float8 FROM reviews WHERE business_id = $1 AND status <> 'hidden' AND created_at >= $2 AND created_at < $3),
  (SELECT count(*) FROM benefit_codes WHERE business_id = $1 AND status = 'redeemed' AND redeemed_at >= $2 AND redeemed_at < $3),
  (SELECT count(*) FROM benefit_codes bc
    WHERE bc.business_id = $1 AND bc.status = 'redeemed' AND bc.redeemed_at >= $2 AND bc.redeemed_at < $3
      AND EXISTS (SELECT 1 FROM reviews r WHERE r.benefit_code_id = bc.id AND r.status <> 'hidden')),
  (SELECT count(*) FROM benefit_codes WHERE business_id = $1 AND created_at >= $2 AND created_at < $3)`,
		businessID, p.From, p.To,
	).Scan(&out.Reviews, &out.Average, &out.Redeemed, &out.WithReview, &out.CouponsGenerated)
	if err != nil {
		return benefits.ReviewStats{}, fmt.Errorf("postgres: review stats: %w", err)
	}
	return out, nil
}
