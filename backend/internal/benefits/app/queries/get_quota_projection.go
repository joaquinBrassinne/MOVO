package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

// GetQuotaProjection answers when the monthly coupon quota would run out at the current pace
// (SDD 6.13), a Pro metric. It has no period: it looks at the month in course, on Argentina's
// calendar. A business without a quota gets status no_quota and nothing else.
type GetQuotaProjection struct {
	gate   ProGate
	quotas benefits.CouponQuotas
	now    func() time.Time
}

// NewGetQuotaProjection wires the query. A nil clock uses time.Now.
func NewGetQuotaProjection(gate ProGate, quotas benefits.CouponQuotas, now func() time.Time) *GetQuotaProjection {
	if now == nil {
		now = time.Now
	}
	return &GetQuotaProjection{gate: gate, quotas: quotas, now: now}
}

// QuotaProjectionResult is the projection with the day the quota resets.
type QuotaProjectionResult struct {
	domain.QuotaProjection
	ResetsOn string
}

// Handle checks the plan first, then projects the quota of the current month.
func (q *GetQuotaProjection) Handle(ctx context.Context, ownerID int64) (QuotaProjectionResult, error) {
	businessID, err := q.gate.Business(ctx, ownerID)
	if err != nil {
		return QuotaProjectionResult{}, err
	}
	now := q.now()
	start, end := domain.QuotaMonth(now)
	quota, err := q.quotas.Quota(ctx, businessID, start, end)
	if err != nil {
		return QuotaProjectionResult{}, fmt.Errorf("read quota: %w", err)
	}
	return QuotaProjectionResult{
		QuotaProjection: domain.ProjectQuota(quota, start, end, now),
		ResetsOn:        domain.DateInArgentina(domain.QuotaResetsOn(now)),
	}, nil
}
