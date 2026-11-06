package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

const countGenerated = `SELECT count(*) FROM benefit_codes WHERE business_id = $1 AND created_at >= $2 AND created_at < $3`

// Quota reads the limit of the business and what it generated in the month.
func (r *Coupons) Quota(ctx context.Context, businessID int64, monthStart, monthEnd time.Time) (domain.Quota, error) {
	var q domain.Quota
	err := r.pool.QueryRow(ctx, `
SELECT monthly_coupon_limit, (`+countGenerated+`)
FROM businesses WHERE id = $1`, businessID, monthStart, monthEnd).Scan(&q.Limit, &q.Generated)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Quota{}, benefits.ErrBusinessNotFound
	}
	if err != nil {
		return domain.Quota{}, fmt.Errorf("postgres: read coupon quota: %w", err)
	}
	return q, nil
}

// SetQuotaLimit sets the limit in one transaction that locks the business row first,
// the same row Issue locks: a coupon being generated finishes before the count is read,
// and one that starts after sees the new limit.
func (r *Coupons) SetQuotaLimit(ctx context.Context, businessID int64, limit *int, monthStart, monthEnd time.Time) (domain.Quota, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Quota{}, fmt.Errorf("postgres: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var locked int64
	if err := tx.QueryRow(ctx, `SELECT id FROM businesses WHERE id = $1 FOR UPDATE`, businessID).Scan(&locked); errors.Is(err, pgx.ErrNoRows) {
		return domain.Quota{}, benefits.ErrBusinessNotFound
	} else if err != nil {
		return domain.Quota{}, fmt.Errorf("postgres: lock business: %w", err)
	}

	var generated int
	if err := tx.QueryRow(ctx, countGenerated, businessID, monthStart, monthEnd).Scan(&generated); err != nil {
		return domain.Quota{}, fmt.Errorf("postgres: count coupons of the month: %w", err)
	}
	if err := domain.ValidateQuotaLimit(limit, generated); err != nil {
		return domain.Quota{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE businesses SET monthly_coupon_limit = $2, updated_at = now() WHERE id = $1`, businessID, limit); err != nil {
		return domain.Quota{}, fmt.Errorf("postgres: set coupon quota: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Quota{}, fmt.Errorf("postgres: commit: %w", err)
	}
	return domain.Quota{Limit: limit, Generated: generated}, nil
}
