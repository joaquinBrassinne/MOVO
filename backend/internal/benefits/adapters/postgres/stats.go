package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

// MonthlyUsage counts, by benefit, the coupons generated and the coupons redeemed in the
// month. The two counts are independent: a coupon generated last month and redeemed this
// month counts only as redeemed.
func (r *Coupons) MonthlyUsage(ctx context.Context, businessID int64, monthStart, monthEnd time.Time) (map[int64]benefits.Usage, error) {
	rows, err := r.pool.Query(ctx, `
SELECT benefit_id,
       count(*) FILTER (WHERE created_at >= $2 AND created_at < $3),
       count(*) FILTER (WHERE status = 'redeemed' AND redeemed_at >= $2 AND redeemed_at < $3)
FROM benefit_codes
WHERE business_id = $1 AND (created_at >= $2 OR redeemed_at >= $2)
GROUP BY benefit_id`, businessID, monthStart, monthEnd)
	if err != nil {
		return nil, fmt.Errorf("postgres: monthly usage: %w", err)
	}
	defer rows.Close()

	out := map[int64]benefits.Usage{}
	for rows.Next() {
		var (
			id int64
			u  benefits.Usage
		)
		if err := rows.Scan(&id, &u.Generated, &u.Redeemed); err != nil {
			return nil, fmt.Errorf("postgres: scan usage: %w", err)
		}
		out[id] = u
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: read usage: %w", err)
	}
	return out, nil
}

// Overview counts the numbers of the panel's home with plain queries. The views and the
// clicks are summed from metric_daily over the last 7 days, whatever the branch or service.
func (r *Coupons) Overview(ctx context.Context, businessID int64, now time.Time) (benefits.Overview, error) {
	dayStart, dayEnd := domain.DayInArgentina(now)
	from := domain.DateInArgentina(dayStart.AddDate(0, 0, -6))
	to := domain.DateInArgentina(dayStart)

	var out benefits.Overview
	err := r.pool.QueryRow(ctx, `
SELECT
  (SELECT count(*) FROM benefit_codes WHERE business_id = $1 AND status = 'pending' AND expires_at > $2),
  (SELECT count(*) FROM benefit_codes WHERE business_id = $1 AND status = 'redeemed' AND redeemed_at >= $3 AND redeemed_at < $4),
  (SELECT COALESCE(sum(count), 0) FROM metric_daily WHERE business_id = $1 AND metric = 'profile_view'   AND date BETWEEN $5::date AND $6::date),
  (SELECT COALESCE(sum(count), 0) FROM metric_daily WHERE business_id = $1 AND metric = 'whatsapp_click' AND date BETWEEN $5::date AND $6::date)`,
		businessID, now, dayStart, dayEnd, from, to,
	).Scan(&out.ActiveCoupons, &out.RedeemedToday, &out.ProfileViews, &out.WhatsAppClicks)
	if err != nil {
		return benefits.Overview{}, fmt.Errorf("postgres: overview: %w", err)
	}
	return out, nil
}
