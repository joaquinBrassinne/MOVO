// Package postgres reads the super admin's figures. It reads the tables of other modules on
// purpose: a report has no business rules to protect, so it asks the database directly.
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/admin"
)

// Overview implements admin.OverviewReader.
type Overview struct {
	pool *pgxpool.Pool
}

// NewOverview builds the reader over pool.
func NewOverview(pool *pgxpool.Pool) *Overview {
	return &Overview{pool: pool}
}

const overviewSQL = `
SELECT
    (SELECT count(*) FROM businesses WHERE status = 'active'),
    count(*),
    count(*) FILTER (WHERE status = 'redeemed'),
    count(*) FILTER (WHERE created_at >= $1),
    count(*) FILTER (WHERE redeemed_at >= $1)
FROM benefit_codes`

// Overview counts the active businesses and the coupons in one round trip.
func (o *Overview) Overview(ctx context.Context, monthStart time.Time) (admin.Overview, error) {
	var out admin.Overview
	err := o.pool.QueryRow(ctx, overviewSQL, monthStart).Scan(
		&out.ActiveBusinesses,
		&out.Coupons.Generated, &out.Coupons.Redeemed,
		&out.CouponsThisMonth.Generated, &out.CouponsThisMonth.Redeemed,
	)
	if err != nil {
		return admin.Overview{}, fmt.Errorf("postgres: read overview: %w", err)
	}
	return out, nil
}
