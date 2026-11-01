package postgres

import (
	"context"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
)

// Essential counts the views of the profile and the coupons generated and redeemed in the
// period. The two coupon counts are independent: one generated before the period and
// redeemed inside it counts only as redeemed.
func (r *Coupons) Essential(ctx context.Context, businessID int64, p benefits.MetricsPeriod) (benefits.EssentialMetrics, error) {
	var out benefits.EssentialMetrics
	err := r.pool.QueryRow(ctx, `
SELECT
  (SELECT COALESCE(sum(count), 0) FROM metric_daily WHERE business_id = $1 AND metric = 'profile_view' AND date BETWEEN $4::date AND $5::date),
  (SELECT count(*) FROM benefit_codes WHERE business_id = $1 AND created_at >= $2 AND created_at < $3),
  (SELECT count(*) FROM benefit_codes WHERE business_id = $1 AND status = 'redeemed' AND redeemed_at >= $2 AND redeemed_at < $3)`,
		businessID, p.From, p.To, p.FromDay, p.ToDay,
	).Scan(&out.ProfileViews, &out.CouponsGenerated, &out.CouponsRedeemed)
	if err != nil {
		return benefits.EssentialMetrics{}, fmt.Errorf("postgres: essential metrics: %w", err)
	}
	return out, nil
}
