package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
)

// Records reads the redemptions per calendar month of the 12 months before monthStart, the
// current month so far, the weekday, slot and benefit of those 12 months, and when the business
// generated its first coupon.
func (r *Coupons) Records(ctx context.Context, businessID int64, windowFrom, monthStart time.Time) (benefits.RecordsData, error) {
	var out benefits.RecordsData

	var first *time.Time
	if err := r.pool.QueryRow(ctx, `SELECT min(created_at) FROM benefit_codes WHERE business_id = $1`, businessID).Scan(&first); err != nil &&
		!errors.Is(err, pgx.ErrNoRows) {
		return benefits.RecordsData{}, fmt.Errorf("postgres: first coupon: %w", err)
	}
	out.FirstCouponAt = first

	monthEnd := monthStart.AddDate(0, 1, 0)
	if err := r.pool.QueryRow(ctx, `
SELECT count(*) FROM benefit_codes
WHERE business_id = $1 AND status = 'redeemed' AND redeemed_at >= $2 AND redeemed_at < $3`,
		businessID, monthStart, monthEnd).Scan(&out.CurrentMonthRedemptions); err != nil {
		return benefits.RecordsData{}, fmt.Errorf("postgres: current month redemptions: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
SELECT to_char(redeemed_at AT TIME ZONE '`+zoneName+`', 'YYYY-MM'), count(*)
FROM benefit_codes
WHERE business_id = $1 AND status = 'redeemed' AND redeemed_at >= $2 AND redeemed_at < $3
GROUP BY 1
ORDER BY 1`, businessID, windowFrom, monthStart)
	if err != nil {
		return benefits.RecordsData{}, fmt.Errorf("postgres: redemptions per month: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var m benefits.MonthRedemptions
		if err := rows.Scan(&m.Month, &m.Redemptions); err != nil {
			return benefits.RecordsData{}, fmt.Errorf("postgres: scan month: %w", err)
		}
		out.Months = append(out.Months, m)
	}
	if err := rows.Err(); err != nil {
		return benefits.RecordsData{}, fmt.Errorf("postgres: read months: %w", err)
	}
	rows.Close()

	if out.Cells, err = r.heatCells(ctx, businessID, windowFrom, monthStart); err != nil {
		return benefits.RecordsData{}, err
	}

	byBenefit, err := r.pool.Query(ctx, `
SELECT benefit_id, count(*)
FROM benefit_codes
WHERE business_id = $1 AND status = 'redeemed' AND redeemed_at >= $2 AND redeemed_at < $3
GROUP BY benefit_id
ORDER BY benefit_id`, businessID, windowFrom, monthStart)
	if err != nil {
		return benefits.RecordsData{}, fmt.Errorf("postgres: redemptions per benefit: %w", err)
	}
	defer byBenefit.Close()
	for byBenefit.Next() {
		var b benefits.BenefitRedemptions
		if err := byBenefit.Scan(&b.BenefitID, &b.Redemptions); err != nil {
			return benefits.RecordsData{}, fmt.Errorf("postgres: scan benefit redemptions: %w", err)
		}
		out.Benefits = append(out.Benefits, b)
	}
	if err := byBenefit.Err(); err != nil {
		return benefits.RecordsData{}, fmt.Errorf("postgres: read benefit redemptions: %w", err)
	}
	return out, nil
}
