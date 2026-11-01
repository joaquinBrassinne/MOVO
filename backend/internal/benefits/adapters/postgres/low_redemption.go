package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
)

// LowRedemption lists the active businesses that generated at least minCoupons coupons in
// [from, to) and got fewer than below of them redeemed (a cohort: redeemed by now). It counts
// per business and returns no coupon or customer.
func (r *Coupons) LowRedemption(ctx context.Context, from, to time.Time, minCoupons int, below float64) ([]benefits.CouponRate, error) {
	rows, err := r.pool.Query(ctx, `
SELECT c.business_id, count(*), count(*) FILTER (WHERE c.status = 'redeemed')
FROM benefit_codes c
JOIN businesses b ON b.id = c.business_id AND b.status = 'active'
WHERE c.created_at >= $1 AND c.created_at < $2
GROUP BY c.business_id
HAVING count(*) >= $3
   AND (count(*) FILTER (WHERE c.status = 'redeemed'))::float8 / count(*) < $4
ORDER BY c.business_id`, from, to, minCoupons, below)
	if err != nil {
		return nil, fmt.Errorf("postgres: low redemption: %w", err)
	}
	defer rows.Close()

	var out []benefits.CouponRate
	for rows.Next() {
		var rate benefits.CouponRate
		if err := rows.Scan(&rate.BusinessID, &rate.Generated, &rate.Redeemed); err != nil {
			return nil, fmt.Errorf("postgres: scan low redemption: %w", err)
		}
		out = append(out, rate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: read low redemption: %w", err)
	}
	return out, nil
}
