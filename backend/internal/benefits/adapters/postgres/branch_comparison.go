package postgres

import (
	"context"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
)

// BranchComparison counts, for every branch, the views of its services, the taps on its WhatsApp
// and the coupons redeemed there, in the period and in the one before it. A retired branch is
// listed only if it had activity; an active one is listed even with none.
func (r *Coupons) BranchComparison(ctx context.Context, businessID int64, current, previous benefits.MetricsPeriod) (benefits.BranchComparison, error) {
	totals, err := r.RedemptionTotals(ctx, businessID, current, previous)
	if err != nil {
		return benefits.BranchComparison{}, err
	}
	out := benefits.BranchComparison{
		CouponsGenerated: totals.CouponsGenerated, PreviousCouponsGenerated: totals.PreviousCouponsGenerated,
	}

	rows, err := r.pool.Query(ctx, `
SELECT b.id, b.name, b.active,
       COALESCE(m.views_now, 0), COALESCE(m.taps_now, 0), COALESCE(d.redeemed_now, 0),
       COALESCE(m.views_before, 0), COALESCE(m.taps_before, 0), COALESCE(d.redeemed_before, 0)
FROM branches b
LEFT JOIN (
    SELECT branch_id,
           sum(count) FILTER (WHERE metric = 'service_view'   AND date BETWEEN $6 AND $7) AS views_now,
           sum(count) FILTER (WHERE metric = 'whatsapp_click' AND date BETWEEN $6 AND $7) AS taps_now,
           sum(count) FILTER (WHERE metric = 'service_view'   AND date BETWEEN $8 AND $9) AS views_before,
           sum(count) FILTER (WHERE metric = 'whatsapp_click' AND date BETWEEN $8 AND $9) AS taps_before
    FROM metric_daily
    WHERE business_id = $1 AND branch_id IS NOT NULL AND metric IN ('service_view', 'whatsapp_click')
      AND date BETWEEN $8 AND $7
    GROUP BY branch_id
) m ON m.branch_id = b.id
LEFT JOIN (
    SELECT redeemed_branch_id,
           count(*) FILTER (WHERE redeemed_at >= $2 AND redeemed_at < $3) AS redeemed_now,
           count(*) FILTER (WHERE redeemed_at >= $4 AND redeemed_at < $5) AS redeemed_before
    FROM benefit_codes
    WHERE business_id = $1 AND status = 'redeemed' AND redeemed_at >= $4 AND redeemed_at < $3
    GROUP BY redeemed_branch_id
) d ON d.redeemed_branch_id = b.id
WHERE b.business_id = $1
  AND (b.active OR m.branch_id IS NOT NULL OR d.redeemed_branch_id IS NOT NULL)
ORDER BY b.name, b.id`,
		businessID, current.From, current.To, previous.From, previous.To,
		current.FromDay, current.ToDay, previous.FromDay, previous.ToDay)
	if err != nil {
		return benefits.BranchComparison{}, fmt.Errorf("postgres: branch comparison: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var a benefits.BranchActivity
		if err := rows.Scan(&a.BranchID, &a.Name, &a.Active,
			&a.Current.ServiceViews, &a.Current.WhatsAppClicks, &a.Current.Redemptions,
			&a.Previous.ServiceViews, &a.Previous.WhatsAppClicks, &a.Previous.Redemptions); err != nil {
			return benefits.BranchComparison{}, fmt.Errorf("postgres: scan branch comparison: %w", err)
		}
		out.Branches = append(out.Branches, a)
	}
	if err := rows.Err(); err != nil {
		return benefits.BranchComparison{}, fmt.Errorf("postgres: read branch comparison: %w", err)
	}
	return out, nil
}
