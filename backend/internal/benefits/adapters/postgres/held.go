package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

// ByToken reads the coupon whose access token has that hash. It is a plain SELECT that
// matches the unique hash, so it can only ever return that one coupon, and it selects no
// phone column.
func (r *Coupons) ByToken(ctx context.Context, tokenHash []byte) (benefits.HeldCoupon, error) {
	var (
		out    benefits.HeldCoupon
		status string
	)
	b := &out.Benefit
	err := r.pool.QueryRow(ctx, `
SELECT c.id, c.code, c.status, c.expires_at, c.redeemed_at, cu.name, bs.name,
       `+benefitColumns("b")+`
FROM benefit_codes c
JOIN customers cu ON cu.id = c.customer_id
JOIN benefits b ON b.id = c.benefit_id
JOIN businesses bs ON bs.id = c.business_id
WHERE c.access_token_hash = $1`, tokenHash,
	).Scan(append([]any{&out.Coupon.ID, &out.Coupon.Code, &status, &out.Coupon.ExpiresAt, &out.RedeemedAt, &out.CustomerName, &out.BusinessName},
		benefitDest(b)...)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return benefits.HeldCoupon{}, benefits.ErrCouponNotFound
	}
	if err != nil {
		return benefits.HeldCoupon{}, fmt.Errorf("postgres: coupon by token: %w", err)
	}

	out.Coupon.Status = domain.Status(status)
	out.Coupon.BenefitID, out.Coupon.BusinessID = b.ID, b.BusinessID
	return out, nil
}
