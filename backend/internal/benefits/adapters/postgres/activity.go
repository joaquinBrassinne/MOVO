package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
)

// RecentActivity merges the two things that happen to a coupon, its generation and its
// redemption, into one list. The count of news is a window over the whole merge, so the
// limit trims the list and never the count. The query selects no phone column.
func (r *Coupons) RecentActivity(ctx context.Context, businessID int64, since *time.Time, limit int) (benefits.Activity, error) {
	rows, err := r.pool.Query(ctx, `
WITH events AS (
    SELECT 'coupon_generated' AS kind, c.created_at AS at, c.customer_id, c.benefit_id
    FROM benefit_codes c WHERE c.business_id = $1
    UNION ALL
    SELECT 'coupon_redeemed', c.redeemed_at, c.customer_id, c.benefit_id
    FROM benefit_codes c WHERE c.business_id = $1 AND c.status = 'redeemed'
)
SELECT e.kind, e.at, cu.name,
       ` + benefitColumns("b") + `,
       count(*) FILTER (WHERE e.at > $2) OVER ()
FROM events e
JOIN customers cu ON cu.id = e.customer_id
JOIN benefits b ON b.id = e.benefit_id
ORDER BY e.at DESC, e.kind
LIMIT $3`, businessID, since, limit)
	if err != nil {
		return benefits.Activity{}, fmt.Errorf("postgres: recent activity: %w", err)
	}
	defer rows.Close()

	var out benefits.Activity
	for rows.Next() {
		var e benefits.ActivityEvent
		b := &e.Benefit
		if err := rows.Scan(append(append([]any{&e.Kind, &e.At, &e.CustomerName}, benefitDest(b)...), &out.NewCount)...); err != nil {
			return benefits.Activity{}, fmt.Errorf("postgres: scan activity: %w", err)
		}
		out.Events = append(out.Events, e)
	}
	if err := rows.Err(); err != nil {
		return benefits.Activity{}, fmt.Errorf("postgres: read activity: %w", err)
	}
	return out, nil
}
