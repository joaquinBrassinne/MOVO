package postgres

import (
	"context"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
)

// TimeToRedeem sorts the coupons redeemed in each period by how long they took, and counts the
// coupons generated in each (the sample).
func (r *Coupons) TimeToRedeem(ctx context.Context, businessID int64, current, previous benefits.MetricsPeriod) (benefits.TimeToRedeem, error) {
	var (
		out benefits.TimeToRedeem
		err error
	)
	if out.Current, out.CouponsGenerated, err = r.redeemTimes(ctx, businessID, current); err != nil {
		return benefits.TimeToRedeem{}, err
	}
	if out.Previous, out.PreviousCouponsGenerated, err = r.redeemTimes(ctx, businessID, previous); err != nil {
		return benefits.TimeToRedeem{}, err
	}
	return out, nil
}

func (r *Coupons) redeemTimes(ctx context.Context, businessID int64, p benefits.MetricsPeriod) (benefits.RedeemTimes, int64, error) {
	const sameDay = `(created_at AT TIME ZONE '` + zoneName + `')::date = (redeemed_at AT TIME ZONE '` + zoneName + `')::date`
	var (
		times     benefits.RedeemTimes
		generated int64
	)
	err := r.pool.QueryRow(ctx, `
SELECT
  count(*) FILTER (WHERE status = 'redeemed' AND redeemed_at >= $2 AND redeemed_at < $3 AND `+sameDay+`),
  count(*) FILTER (WHERE status = 'redeemed' AND redeemed_at >= $2 AND redeemed_at < $3 AND NOT `+sameDay+`
                     AND redeemed_at - created_at <= interval '24 hours'),
  count(*) FILTER (WHERE status = 'redeemed' AND redeemed_at >= $2 AND redeemed_at < $3 AND NOT `+sameDay+`
                     AND redeemed_at - created_at > interval '24 hours'),
  (SELECT count(*) FROM benefit_codes WHERE business_id = $1 AND created_at >= $2 AND created_at < $3)
FROM benefit_codes
WHERE business_id = $1 AND status = 'redeemed' AND redeemed_at >= $2 AND redeemed_at < $3`,
		businessID, p.From, p.To,
	).Scan(&times.SameDay, &times.Within24, &times.Within48, &generated)
	if err != nil {
		return benefits.RedeemTimes{}, 0, fmt.Errorf("postgres: time to redeem: %w", err)
	}
	return times, generated, nil
}
