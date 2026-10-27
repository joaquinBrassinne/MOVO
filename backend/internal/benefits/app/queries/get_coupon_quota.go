package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

// GetCouponQuota feeds the "Cupo del mes" card of the owner's panel.
type GetCouponQuota struct {
	owners benefits.OwnerBusinesses
	quotas benefits.CouponQuotas
	now    func() time.Time
}

// NewGetCouponQuota wires the GetCouponQuota query. A nil clock uses time.Now.
func NewGetCouponQuota(owners benefits.OwnerBusinesses, quotas benefits.CouponQuotas, now func() time.Time) *GetCouponQuota {
	if now == nil {
		now = time.Now
	}
	return &GetCouponQuota{owners: owners, quotas: quotas, now: now}
}

// Handle returns the quota of the owner's business for the current month.
func (q *GetCouponQuota) Handle(ctx context.Context, ownerID int64) (benefits.QuotaView, error) {
	businessID, err := q.owners.EditableBusinessOf(ctx, ownerID)
	if err != nil {
		return benefits.QuotaView{}, fmt.Errorf("resolve business: %w", err)
	}
	now := q.now()
	start, end := domain.QuotaMonth(now)
	quota, err := q.quotas.Quota(ctx, businessID, start, end)
	if err != nil {
		return benefits.QuotaView{}, fmt.Errorf("read quota: %w", err)
	}
	return benefits.QuotaView{Quota: quota, ResetsOn: domain.QuotaResetsOn(now)}, nil
}
