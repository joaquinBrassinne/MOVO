package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

// Preview reads one coupon of the business. It is a plain SELECT scoped by business_id,
// so the code of another business is not found, and it selects no phone column at all.
func (r *Coupons) Preview(ctx context.Context, businessID int64, code string) (benefits.CouponPreview, error) {
	var (
		out    benefits.CouponPreview
		status string
	)
	b := &out.Benefit
	err := r.pool.QueryRow(ctx, `
SELECT c.id, c.code, c.status, c.expires_at, c.redeemed_at, cu.name,
       (SELECT count(*) FROM benefit_codes r WHERE r.customer_id = c.customer_id AND r.business_id = c.business_id AND r.status = 'redeemed'),
       `+couponBenefitColumns("c", "b")+`
FROM benefit_codes c
JOIN customers cu ON cu.id = c.customer_id
JOIN benefits b ON b.id = c.benefit_id
WHERE c.code = $1 AND c.business_id = $2`, code, businessID,
	).Scan(append([]any{&out.Coupon.ID, &out.Coupon.Code, &status, &out.Coupon.ExpiresAt, &out.RedeemedAt, &out.CustomerName, &out.CustomerRedeemed},
		benefitDest(b)...)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return benefits.CouponPreview{}, benefits.ErrCouponNotFound
	}
	if err != nil {
		return benefits.CouponPreview{}, fmt.Errorf("postgres: preview coupon: %w", err)
	}

	out.Coupon.Status = domain.Status(status)
	out.Coupon.BenefitID, out.Coupon.BusinessID = b.ID, businessID
	return out, nil
}
