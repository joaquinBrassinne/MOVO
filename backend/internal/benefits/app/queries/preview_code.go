package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

// CouponView is a coupon as the preview shows it.
type CouponView struct {
	Code             string
	Status           domain.Status
	Benefit          domain.Benefit
	CustomerName     string
	CustomerRedeemed int
	ExpiresAt        time.Time
	RedeemedAt       *time.Time
}

// PreviewCode lets an owner look at a coupon of their business before redeeming it
// (SDD 5.2). Scanning a QR or typing a code only gets here: the redemption is a separate
// command, and the answer may be stale by the time it is confirmed.
type PreviewCode struct {
	owners  benefits.OwnerBusinesses
	coupons benefits.CouponReader
	now     func() time.Time
}

// NewPreviewCode wires the PreviewCode query. A nil clock uses time.Now.
func NewPreviewCode(owners benefits.OwnerBusinesses, coupons benefits.CouponReader, now func() time.Time) *PreviewCode {
	if now == nil {
		now = time.Now
	}
	return &PreviewCode{owners: owners, coupons: coupons, now: now}
}

// Handle returns the coupon. It has no side effects: an overdue coupon is shown as
// expired without being marked, because marking is the daily sweep's job. The business
// is the owner's, from the session; a coupon of another business, a code that does not
// exist and a malformed one all answer benefits.ErrCouponNotFound.
func (q *PreviewCode) Handle(ctx context.Context, ownerID int64, rawCode string) (CouponView, error) {
	businessID, err := q.owners.EditableBusinessOf(ctx, ownerID)
	if err != nil {
		return CouponView{}, fmt.Errorf("resolve business: %w", err)
	}
	code, ok := domain.NormalizeCode(rawCode)
	if !ok {
		return CouponView{}, benefits.ErrCouponNotFound
	}

	preview, err := q.coupons.Preview(ctx, businessID, code)
	if err != nil {
		return CouponView{}, fmt.Errorf("read coupon: %w", err)
	}
	return CouponView{
		Code:             preview.Coupon.Code,
		Status:           preview.Coupon.StatusAt(q.now()),
		Benefit:          preview.Benefit,
		CustomerName:     preview.CustomerName,
		CustomerRedeemed: preview.CustomerRedeemed,
		ExpiresAt:        preview.Coupon.ExpiresAt,
		RedeemedAt:       preview.RedeemedAt,
	}, nil
}
