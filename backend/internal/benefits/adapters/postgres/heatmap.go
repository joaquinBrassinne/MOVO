package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
)

// zoneName is the zone the pilot lives in: weekdays and hours are read on its clock.
const zoneName = "America/Argentina/Cordoba"

// Heatmap counts the redemptions of the period by weekday and six-hour slot, the redemptions
// of the period before, and the coupons generated in the period (the sample).
func (r *Coupons) Heatmap(ctx context.Context, businessID int64, current, previous benefits.MetricsPeriod) (benefits.Heatmap, error) {
	var out benefits.Heatmap
	err := r.pool.QueryRow(ctx, `
SELECT
  (SELECT count(*) FROM benefit_codes WHERE business_id = $1 AND created_at >= $2 AND created_at < $3),
  (SELECT count(*) FROM benefit_codes WHERE business_id = $1 AND status = 'redeemed' AND redeemed_at >= $2 AND redeemed_at < $3),
  (SELECT count(*) FROM benefit_codes WHERE business_id = $1 AND status = 'redeemed' AND redeemed_at >= $4 AND redeemed_at < $5)`,
		businessID, current.From, current.To, previous.From, previous.To,
	).Scan(&out.CouponsGenerated, &out.Redemptions, &out.PreviousRedemptions)
	if err != nil {
		return benefits.Heatmap{}, fmt.Errorf("postgres: heatmap totals: %w", err)
	}

	if out.Cells, err = r.heatCells(ctx, businessID, current.From, current.To); err != nil {
		return benefits.Heatmap{}, err
	}
	return out, nil
}

// heatCells counts the coupons redeemed in [from, to) by weekday and six-hour slot on
// Argentina's clock. Only the cells with redemptions are returned.
func (r *Coupons) heatCells(ctx context.Context, businessID int64, from, to time.Time) ([]benefits.HeatCell, error) {
	rows, err := r.pool.Query(ctx, `
SELECT extract(isodow FROM redeemed_at AT TIME ZONE '`+zoneName+`')::int,
       (extract(hour FROM redeemed_at AT TIME ZONE '`+zoneName+`')::int / 6),
       count(*)
FROM benefit_codes
WHERE business_id = $1 AND status = 'redeemed' AND redeemed_at >= $2 AND redeemed_at < $3
GROUP BY 1, 2
ORDER BY 1, 2`, businessID, from, to)
	if err != nil {
		return nil, fmt.Errorf("postgres: heatmap cells: %w", err)
	}
	defer rows.Close()

	var cells []benefits.HeatCell
	for rows.Next() {
		var c benefits.HeatCell
		if err := rows.Scan(&c.Weekday, &c.Slot, &c.Redemptions); err != nil {
			return nil, fmt.Errorf("postgres: scan heatmap cell: %w", err)
		}
		cells = append(cells, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: read heatmap cells: %w", err)
	}
	return cells, nil
}
