package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

// HeldCouponView is a coupon as its customer sees it.
type HeldCouponView struct {
	Code         string
	Status       domain.Status
	Benefit      domain.Benefit
	BusinessName string
	ExpiresAt    time.Time
	RedeemedAt   *time.Time
}

// ViewMyCoupon lets a customer, who has no account, look at the coupon their access
// token opens (SDD 5.2).
type ViewMyCoupon struct {
	coupons benefits.CouponReader
	now     func() time.Time
}

// NewViewMyCoupon wires the ViewMyCoupon query. A nil clock uses time.Now.
func NewViewMyCoupon(coupons benefits.CouponReader, now func() time.Time) *ViewMyCoupon {
	if now == nil {
		now = time.Now
	}
	return &ViewMyCoupon{coupons: coupons, now: now}
}

// Handle returns the coupon of the token. Only the hash reaches the store. It has no side
// effects: an overdue coupon is shown as expired without being marked. An unknown token
// answers benefits.ErrCouponNotFound whatever it looks like.
func (q *ViewMyCoupon) Handle(ctx context.Context, token string) (HeldCouponView, error) {
	held, err := q.coupons.ByToken(ctx, domain.HashAccessToken(token))
	if err != nil {
		return HeldCouponView{}, fmt.Errorf("read coupon: %w", err)
	}
	return HeldCouponView{
		Code:         held.Coupon.Code,
		Status:       held.Coupon.StatusAt(q.now()),
		Benefit:      held.Benefit,
		BusinessName: held.BusinessName,
		ExpiresAt:    held.Coupon.ExpiresAt,
		RedeemedAt:   held.RedeemedAt,
	}, nil
}
