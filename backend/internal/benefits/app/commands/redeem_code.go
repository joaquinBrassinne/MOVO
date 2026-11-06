package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

// RedeemCodeInput is a coupon an owner redeems at the counter.
type RedeemCodeInput struct {
	OwnerID int64
	Code    string
	// BranchID is where it was redeemed; 0 means the only active branch of the business.
	BranchID int64
}

// RedeemCode redeems a coupon of the owner's business.
type RedeemCode struct {
	owners   benefits.OwnerBusinesses
	redeemer benefits.CouponRedeemer
	now      func() time.Time
}

// NewRedeemCode wires the RedeemCode use case. A nil clock uses time.Now.
func NewRedeemCode(owners benefits.OwnerBusinesses, redeemer benefits.CouponRedeemer, now func() time.Time) *RedeemCode {
	if now == nil {
		now = time.Now
	}
	return &RedeemCode{owners: owners, redeemer: redeemer, now: now}
}

// Handle redeems the coupon. The business is the owner's, taken from the session: a
// coupon of another business answers benefits.ErrCouponNotFound, exactly like a code
// that does not exist. The atomic UPDATE in the redeemer is the only thing that
// redeems; nothing read before it can.
func (c *RedeemCode) Handle(ctx context.Context, in RedeemCodeInput) (benefits.Redeemed, error) {
	businessID, err := c.owners.EditableBusinessOf(ctx, in.OwnerID)
	if err != nil {
		return benefits.Redeemed{}, fmt.Errorf("resolve business: %w", err)
	}
	code, ok := domain.NormalizeCode(in.Code)
	if !ok {
		return benefits.Redeemed{}, benefits.ErrCouponNotFound
	}

	redeemed, err := c.redeemer.Redeem(ctx, benefits.RedeemRequest{
		BusinessID: businessID,
		UserID:     in.OwnerID,
		Code:       code,
		BranchID:   in.BranchID,
		Now:        c.now(),
	})
	if err != nil {
		return benefits.Redeemed{}, fmt.Errorf("redeem coupon: %w", err)
	}
	return redeemed, nil
}
