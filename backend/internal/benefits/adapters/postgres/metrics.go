package postgres

import (
	"context"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
)

// TopActivities ranks the services of the business by the views plus the WhatsApp taps they
// got in the period. Taps with no service are not an activity and are left out.
func (r *Coupons) TopActivities(ctx context.Context, businessID int64, p benefits.MetricsPeriod, limit int) ([]benefits.ActivityDemand, error) {
	rows, err := r.pool.Query(ctx, `
SELECT s.id, s.name,
       COALESCE(sum(m.count) FILTER (WHERE m.metric = 'service_view'), 0),
       COALESCE(sum(m.count) FILTER (WHERE m.metric = 'whatsapp_click'), 0)
FROM metric_daily m
JOIN services s ON s.id = m.service_id AND s.business_id = m.business_id
WHERE m.business_id = $1 AND m.service_id IS NOT NULL AND s.deleted_at IS NULL
  AND m.metric IN ('service_view', 'whatsapp_click') AND m.date BETWEEN $2::date AND $3::date
GROUP BY s.id, s.name
ORDER BY sum(m.count) DESC, s.name, s.id
LIMIT $4`, businessID, p.FromDay, p.ToDay, limit)
	if err != nil {
		return nil, fmt.Errorf("postgres: top activities: %w", err)
	}
	defer rows.Close()

	var out []benefits.ActivityDemand
	for rows.Next() {
		var a benefits.ActivityDemand
		if err := rows.Scan(&a.ServiceID, &a.Name, &a.Views, &a.Clicks); err != nil {
			return nil, fmt.Errorf("postgres: scan activity: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: read activities: %w", err)
	}
	return out, nil
}

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
