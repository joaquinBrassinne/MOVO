package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

// SetCouponQuota changes the monthly cap on the coupons of the owner's business.
type SetCouponQuota struct {
	owners benefits.OwnerBusinesses
	quotas benefits.CouponQuotas
	now    func() time.Time
}

// NewSetCouponQuota wires the SetCouponQuota use case. A nil clock uses time.Now.
func NewSetCouponQuota(owners benefits.OwnerBusinesses, quotas benefits.CouponQuotas, now func() time.Time) *SetCouponQuota {
	if now == nil {
		now = time.Now
	}
	return &SetCouponQuota{owners: owners, quotas: quotas, now: now}
}

// Handle sets the limit (nil removes it). The business is the owner's, from the session.
// A limit that is not greater than zero is refused before touching the store; one below
// what was already generated this month is refused by the store, which sees the count
// under the same lock as the generation of coupons.
func (c *SetCouponQuota) Handle(ctx context.Context, ownerID int64, limit *int) (benefits.QuotaView, error) {
	if err := domain.ValidateQuotaLimit(limit, 0); err != nil {
		return benefits.QuotaView{}, err
	}
	businessID, err := c.owners.EditableBusinessOf(ctx, ownerID)
	if err != nil {
		return benefits.QuotaView{}, fmt.Errorf("resolve business: %w", err)
	}
	now := c.now()
	start, end := domain.QuotaMonth(now)
	quota, err := c.quotas.SetQuotaLimit(ctx, businessID, limit, start, end)
	if err != nil {
		return benefits.QuotaView{}, fmt.Errorf("set quota: %w", err)
	}
	return benefits.QuotaView{Quota: quota, ResetsOn: domain.QuotaResetsOn(now)}, nil
}
