package postgres

import (
	"context"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
)

// RedemptionTotals counts, for each of the two periods, the coupons redeemed in it and the
// coupons generated in it.
func (r *Coupons) RedemptionTotals(ctx context.Context, businessID int64, current, previous benefits.MetricsPeriod) (benefits.RedemptionTotals, error) {
	var (
		out benefits.RedemptionTotals
		err error
	)
	if out.Redemptions, out.CouponsGenerated, err = r.windowTotals(ctx, businessID, current); err != nil {
		return benefits.RedemptionTotals{}, err
	}
	if out.PreviousRedemptions, out.PreviousCouponsGenerated, err = r.windowTotals(ctx, businessID, previous); err != nil {
		return benefits.RedemptionTotals{}, err
	}
	return out, nil
}

func (r *Coupons) windowTotals(ctx context.Context, businessID int64, p benefits.MetricsPeriod) (redemptions, generated int64, err error) {
	err = r.pool.QueryRow(ctx, `
SELECT
  (SELECT count(*) FROM benefit_codes WHERE business_id = $1 AND status = 'redeemed' AND redeemed_at >= $2 AND redeemed_at < $3),
  (SELECT count(*) FROM benefit_codes WHERE business_id = $1 AND created_at >= $2 AND created_at < $3)`,
		businessID, p.From, p.To).Scan(&redemptions, &generated)
	if err != nil {
		return 0, 0, fmt.Errorf("postgres: redemption totals: %w", err)
	}
	return redemptions, generated, nil
}
