package postgres

import (
	"context"
	"fmt"
	"time"
)

// ExpireOverdue is one conditional UPDATE: only a pending coupon past its time changes,
// so a coupon redeemed a moment earlier stays redeemed, and the same bound that
// Redeem uses (expires_at > now means still valid) decides who is overdue.
func (r *Coupons) ExpireOverdue(ctx context.Context, now time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE benefit_codes SET status = 'expired' WHERE status = 'pending' AND expires_at <= $1`, now)
	if err != nil {
		return 0, fmt.Errorf("postgres: expire overdue coupons: %w", err)
	}
	return tag.RowsAffected(), nil
}
