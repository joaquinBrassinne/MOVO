package postgres

import (
	"context"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
)

// RepeatCustomers counts the customers who redeemed once and the ones who redeemed two or more
// times in each period, and the coupons generated in each. The grouping by customer happens in
// the subquery: only the two counts leave the database, never a customer.
func (r *Coupons) RepeatCustomers(ctx context.Context, businessID int64, current, previous benefits.MetricsPeriod) (benefits.RepeatCustomers, error) {
	var (
		out benefits.RepeatCustomers
		err error
	)
	if out.Current, out.CouponsGenerated, err = r.customerCounts(ctx, businessID, current); err != nil {
		return benefits.RepeatCustomers{}, err
	}
	if out.Previous, out.PreviousCouponsGenerated, err = r.customerCounts(ctx, businessID, previous); err != nil {
		return benefits.RepeatCustomers{}, err
	}
	return out, nil
}

func (r *Coupons) customerCounts(ctx context.Context, businessID int64, p benefits.MetricsPeriod) (benefits.CustomerCounts, int64, error) {
	var (
		counts    benefits.CustomerCounts
		generated int64
	)
	err := r.pool.QueryRow(ctx, `
SELECT
  count(*) FILTER (WHERE redemptions = 1),
  count(*) FILTER (WHERE redemptions >= 2),
  (SELECT count(*) FROM benefit_codes WHERE business_id = $1 AND created_at >= $2 AND created_at < $3)
FROM (
    SELECT count(*) AS redemptions
    FROM benefit_codes
    WHERE business_id = $1 AND status = 'redeemed' AND redeemed_at >= $2 AND redeemed_at < $3
    GROUP BY customer_id
) per_customer`, businessID, p.From, p.To,
	).Scan(&counts.OneRedemption, &counts.Repeat, &generated)
	if err != nil {
		return benefits.CustomerCounts{}, 0, fmt.Errorf("postgres: repeat customers: %w", err)
	}
	return counts, generated, nil
}
