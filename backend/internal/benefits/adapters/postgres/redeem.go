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

// Redeem redeems a coupon with one conditional UPDATE (SDD 5.2). The WHERE clause is the
// whole rule: the code, the business of the session, still pending and not past its
// expiry. Two simultaneous redemptions of one code cannot both match, because the second
// UPDATE waits for the first and then no longer finds a pending row.
func (r *Coupons) Redeem(ctx context.Context, req benefits.RedeemRequest) (benefits.Redeemed, error) {
	branchID, err := r.redeemBranch(ctx, req.BusinessID, req.BranchID)
	if err != nil {
		return benefits.Redeemed{}, err
	}

	var (
		out      benefits.Redeemed
		benefit  domain.Benefit
		redeemed time.Time
	)
	err = r.pool.QueryRow(ctx, `
WITH redeemed AS (
    UPDATE benefit_codes
    SET status = 'redeemed', redeemed_at = $3, redeemed_by = $4, redeemed_branch_id = $5, updated_at = now()
    WHERE code = $1 AND business_id = $2 AND status = 'pending' AND expires_at > $3
    RETURNING id, code, benefit_id, business_id, customer_id, expires_at, redeemed_at, `+couponTermsColumns+`
)
SELECT r.id, r.code, r.expires_at, r.redeemed_at, cu.name,
       `+couponBenefitColumns("r", "b")+`
FROM redeemed r
JOIN customers cu ON cu.id = r.customer_id
JOIN benefits b ON b.id = r.benefit_id`,
		req.Code, req.BusinessID, req.Now, req.UserID, branchID,
	).Scan(append([]any{&out.Coupon.ID, &out.Coupon.Code, &out.Coupon.ExpiresAt, &redeemed, &out.CustomerName},
		benefitDest(&benefit)...)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return benefits.Redeemed{}, r.whyNotRedeemed(ctx, req)
	}
	if err != nil {
		return benefits.Redeemed{}, fmt.Errorf("postgres: redeem coupon: %w", err)
	}

	out.Coupon.Status = domain.StatusRedeemed
	out.Coupon.BenefitID, out.Coupon.BusinessID = benefit.ID, req.BusinessID
	out.Benefit, out.BranchID, out.RedeemedAt = benefit, branchID, redeemed
	return out, nil
}

// whyNotRedeemed tells the owner why the UPDATE matched nothing. It only reads coupons
// of the owner's own business, so another business's code is "not found" like an unknown
// one. The answer is informative: it never decides anything.
func (r *Coupons) whyNotRedeemed(ctx context.Context, req benefits.RedeemRequest) error {
	var (
		status  string
		expires time.Time
	)
	err := r.pool.QueryRow(ctx,
		`SELECT status, expires_at FROM benefit_codes WHERE code = $1 AND business_id = $2`,
		req.Code, req.BusinessID).Scan(&status, &expires)
	if errors.Is(err, pgx.ErrNoRows) {
		return benefits.ErrCouponNotFound
	}
	if err != nil {
		return fmt.Errorf("postgres: read coupon: %w", err)
	}

	switch {
	case domain.Status(status) == domain.StatusRedeemed:
		return benefits.ErrCouponRedeemed
	case domain.Status(status) == domain.StatusExpired, !req.Now.Before(expires):
		return benefits.ErrCouponExpired
	}
	// Pending and in time: another request redeemed it between the UPDATE and this read.
	return benefits.ErrCouponRedeemed
}

// redeemBranch resolves where the redemption happened: the given branch, which must be
// an active one of the business, or the only active branch when none is given.
func (r *Coupons) redeemBranch(ctx context.Context, businessID, branchID int64) (int64, error) {
	if branchID != 0 {
		var id int64
		err := r.pool.QueryRow(ctx, `SELECT id FROM branches WHERE id = $1 AND business_id = $2 AND active`, branchID, businessID).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, benefits.ErrBranchNotFound
		}
		if err != nil {
			return 0, fmt.Errorf("postgres: find branch: %w", err)
		}
		return id, nil
	}

	rows, err := r.pool.Query(ctx, `SELECT id FROM branches WHERE business_id = $1 AND active ORDER BY id LIMIT 2`, businessID)
	if err != nil {
		return 0, fmt.Errorf("postgres: list branches: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return 0, fmt.Errorf("postgres: scan branch: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("postgres: list branches: %w", err)
	}
	if len(ids) != 1 {
		return 0, benefits.ErrBranchRequired
	}
	return ids[0], nil
}
