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
		out        benefits.CouponPreview
		status     string
		customerID int64
	)
	b := &out.Benefit
	err := r.pool.QueryRow(ctx, `
SELECT c.id, c.code, c.status, c.expires_at, c.redeemed_at, cu.name, c.customer_id,
       (SELECT count(*) FROM benefit_codes r WHERE r.customer_id = c.customer_id AND r.business_id = c.business_id AND r.status = 'redeemed'),
       `+couponBenefitColumns("c", "b")+`
FROM benefit_codes c
JOIN customers cu ON cu.id = c.customer_id
JOIN benefits b ON b.id = c.benefit_id
WHERE c.code = $1 AND c.business_id = $2`, code, businessID,
	).Scan(append([]any{&out.Coupon.ID, &out.Coupon.Code, &status, &out.Coupon.ExpiresAt, &out.RedeemedAt, &out.CustomerName, &customerID, &out.CustomerRedeemed},
		benefitDest(b)...)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return benefits.CouponPreview{}, benefits.ErrCouponNotFound
	}
	if err != nil {
		return benefits.CouponPreview{}, fmt.Errorf("postgres: preview coupon: %w", err)
	}

	out.Coupon.Status = domain.Status(status)
	out.Coupon.BenefitID, out.Coupon.BusinessID = b.ID, businessID

	if out.PreviousRedemptions, err = r.previousRedemptions(ctx, businessID, customerID, out.Coupon.ID); err != nil {
		return benefits.CouponPreview{}, err
	}
	return out, nil
}

// previousRedemptions lists the newest coupons this customer redeemed at the business,
// the one being previewed left out. It is informative, like the rest of the preview: it
// decides nothing, and it selects no phone column.
func (r *Coupons) previousRedemptions(ctx context.Context, businessID, customerID, exceptCouponID int64) ([]benefits.PastRedemption, error) {
	rows, err := r.pool.Query(ctx, `
SELECT c.redeemed_at, br.name, `+couponBenefitColumns("c", "b")+`
FROM benefit_codes c
JOIN benefits b ON b.id = c.benefit_id
JOIN branches br ON br.id = c.redeemed_branch_id
WHERE c.business_id = $1 AND c.customer_id = $2 AND c.status = 'redeemed' AND c.id <> $3
ORDER BY c.redeemed_at DESC, c.id DESC
LIMIT $4`, businessID, customerID, exceptCouponID, benefits.PreviousRedemptionsLimit)
	if err != nil {
		return nil, fmt.Errorf("postgres: previous redemptions: %w", err)
	}
	defer rows.Close()

	past := []benefits.PastRedemption{}
	for rows.Next() {
		var p benefits.PastRedemption
		if err := rows.Scan(append([]any{&p.RedeemedAt, &p.BranchName}, benefitDest(&p.Benefit)...)...); err != nil {
			return nil, fmt.Errorf("postgres: scan previous redemption: %w", err)
		}
		past = append(past, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: read previous redemptions: %w", err)
	}
	return past, nil
}
