package postgres

import (
	"context"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
)

// Conversion reads the counts of the period, the counts of the one before it and the
// breakdown of the period by branch and by activity.
func (r *Coupons) Conversion(ctx context.Context, businessID int64, current, previous benefits.MetricsPeriod) (benefits.Conversion, error) {
	var (
		out benefits.Conversion
		err error
	)
	if out.Current, err = r.conversionCounts(ctx, businessID, current); err != nil {
		return benefits.Conversion{}, err
	}
	if out.Previous, err = r.conversionCounts(ctx, businessID, previous); err != nil {
		return benefits.Conversion{}, err
	}
	if out.ByBranch, err = r.conversionByBranch(ctx, businessID, current); err != nil {
		return benefits.Conversion{}, err
	}
	if out.ByActivity, err = r.conversionByActivity(ctx, businessID, current); err != nil {
		return benefits.Conversion{}, err
	}
	return out, nil
}

// conversionCounts counts the views of the profile and the cohort of coupons generated in the
// period, and how many of them have been redeemed by now.
func (r *Coupons) conversionCounts(ctx context.Context, businessID int64, p benefits.MetricsPeriod) (benefits.ConversionCounts, error) {
	var out benefits.ConversionCounts
	err := r.pool.QueryRow(ctx, `
SELECT
  (SELECT COALESCE(sum(count), 0) FROM metric_daily
    WHERE business_id = $1 AND metric = 'profile_view' AND date BETWEEN $4::date AND $5::date),
  count(*),
  count(*) FILTER (WHERE status = 'redeemed')
FROM benefit_codes
WHERE business_id = $1 AND created_at >= $2 AND created_at < $3`,
		businessID, p.From, p.To, p.FromDay, p.ToDay,
	).Scan(&out.ProfileViews, &out.CouponsGenerated, &out.CouponsRedeemed)
	if err != nil {
		return benefits.ConversionCounts{}, fmt.Errorf("postgres: conversion counts: %w", err)
	}
	return out, nil
}

// conversionByBranch lists the branches that got a WhatsApp tap or a redemption in the period.
// Retired branches stay: their history is still the business's.
func (r *Coupons) conversionByBranch(ctx context.Context, businessID int64, p benefits.MetricsPeriod) ([]benefits.BranchConversion, error) {
	rows, err := r.pool.Query(ctx, `
SELECT b.id, b.name, COALESCE(c.clicks, 0), COALESCE(d.redemptions, 0)
FROM branches b
LEFT JOIN (
    SELECT branch_id, sum(count) AS clicks FROM metric_daily
    WHERE business_id = $1 AND metric = 'whatsapp_click' AND branch_id IS NOT NULL AND date BETWEEN $4::date AND $5::date
    GROUP BY branch_id
) c ON c.branch_id = b.id
LEFT JOIN (
    SELECT redeemed_branch_id, count(*) AS redemptions FROM benefit_codes
    WHERE business_id = $1 AND status = 'redeemed' AND redeemed_at >= $2 AND redeemed_at < $3
    GROUP BY redeemed_branch_id
) d ON d.redeemed_branch_id = b.id
WHERE b.business_id = $1 AND (c.clicks IS NOT NULL OR d.redemptions IS NOT NULL)
ORDER BY b.name, b.id`, businessID, p.From, p.To, p.FromDay, p.ToDay)
	if err != nil {
		return nil, fmt.Errorf("postgres: conversion by branch: %w", err)
	}
	defer rows.Close()

	var out []benefits.BranchConversion
	for rows.Next() {
		var b benefits.BranchConversion
		if err := rows.Scan(&b.BranchID, &b.Name, &b.WhatsAppClicks, &b.Redemptions); err != nil {
			return nil, fmt.Errorf("postgres: scan branch conversion: %w", err)
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: read branch conversion: %w", err)
	}
	return out, nil
}

// conversionByActivity lists the services that were viewed or tapped in the period. Services
// the owner deleted are left out, like everywhere on the owner's side.
func (r *Coupons) conversionByActivity(ctx context.Context, businessID int64, p benefits.MetricsPeriod) ([]benefits.ActivityConversion, error) {
	rows, err := r.pool.Query(ctx, `
SELECT s.id, s.name,
       COALESCE(sum(m.count) FILTER (WHERE m.metric = 'service_view'), 0),
       COALESCE(sum(m.count) FILTER (WHERE m.metric = 'whatsapp_click'), 0)
FROM metric_daily m
JOIN services s ON s.id = m.service_id AND s.business_id = m.business_id
WHERE m.business_id = $1 AND m.service_id IS NOT NULL AND s.deleted_at IS NULL
  AND m.metric IN ('service_view', 'whatsapp_click') AND m.date BETWEEN $2::date AND $3::date
GROUP BY s.id, s.name
ORDER BY s.name, s.id`, businessID, p.FromDay, p.ToDay)
	if err != nil {
		return nil, fmt.Errorf("postgres: conversion by activity: %w", err)
	}
	defer rows.Close()

	var out []benefits.ActivityConversion
	for rows.Next() {
		var a benefits.ActivityConversion
		if err := rows.Scan(&a.ServiceID, &a.Name, &a.Views, &a.WhatsAppClicks); err != nil {
			return nil, fmt.Errorf("postgres: scan activity conversion: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: read activity conversion: %w", err)
	}
	return out, nil
}
