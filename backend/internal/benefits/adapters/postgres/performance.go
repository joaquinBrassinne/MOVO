package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
)

// BenefitPerformance counts, per benefit, the coupons generated in each period and what became
// of them: redeemed, expired (marked, or pending past their time), or still pending.
func (r *Coupons) BenefitPerformance(ctx context.Context, businessID int64, current, previous benefits.MetricsPeriod, now time.Time) (benefits.Performance, error) {
	var (
		out benefits.Performance
		err error
	)
	if out.Current, err = r.benefitCounts(ctx, businessID, current, now); err != nil {
		return benefits.Performance{}, err
	}
	if out.Previous, err = r.benefitCounts(ctx, businessID, previous, now); err != nil {
		return benefits.Performance{}, err
	}
	return out, nil
}

func (r *Coupons) benefitCounts(ctx context.Context, businessID int64, p benefits.MetricsPeriod, now time.Time) ([]benefits.BenefitCounts, error) {
	rows, err := r.pool.Query(ctx, `
SELECT benefit_id,
       count(*),
       count(*) FILTER (WHERE status = 'redeemed'),
       count(*) FILTER (WHERE status = 'expired' OR (status = 'pending' AND expires_at <= $4))
FROM benefit_codes
WHERE business_id = $1 AND created_at >= $2 AND created_at < $3
GROUP BY benefit_id
ORDER BY benefit_id`, businessID, p.From, p.To, now)
	if err != nil {
		return nil, fmt.Errorf("postgres: benefit performance: %w", err)
	}
	defer rows.Close()

	var out []benefits.BenefitCounts
	for rows.Next() {
		var c benefits.BenefitCounts
		if err := rows.Scan(&c.BenefitID, &c.Generated, &c.Redeemed, &c.Expired); err != nil {
			return nil, fmt.Errorf("postgres: scan benefit performance: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: read benefit performance: %w", err)
	}
	return out, nil
}
