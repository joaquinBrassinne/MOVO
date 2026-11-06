// Package queries holds the read-only benefit use cases.
package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

// ListMyBenefits returns the benefits of the owner's business for the panel, each with
// what it did this month.
type ListMyBenefits struct {
	owners   benefits.OwnerBusinesses
	benefits benefits.BenefitRepository
	usage    benefits.BenefitUsage
	now      func() time.Time
}

// NewListMyBenefits wires the ListMyBenefits query. A nil clock uses time.Now.
func NewListMyBenefits(owners benefits.OwnerBusinesses, repo benefits.BenefitRepository, usage benefits.BenefitUsage, now func() time.Time) *ListMyBenefits {
	if now == nil {
		now = time.Now
	}
	return &ListMyBenefits{owners: owners, benefits: repo, usage: usage, now: now}
}

// Handle lists every benefit, the inactive ones included, oldest first. The usage is of
// the current calendar month in Argentina; a benefit with no coupons has zeros.
func (q *ListMyBenefits) Handle(ctx context.Context, ownerID int64) ([]benefits.BenefitWithUsage, error) {
	businessID, err := q.owners.EditableBusinessOf(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("resolve business: %w", err)
	}
	list, err := q.benefits.List(ctx, businessID)
	if err != nil {
		return nil, fmt.Errorf("list benefits: %w", err)
	}
	start, end := domain.QuotaMonth(q.now())
	usage, err := q.usage.MonthlyUsage(ctx, businessID, start, end)
	if err != nil {
		return nil, fmt.Errorf("read usage: %w", err)
	}

	out := make([]benefits.BenefitWithUsage, 0, len(list))
	for _, b := range list {
		out = append(out, benefits.BenefitWithUsage{Benefit: b, Usage: usage[b.ID]})
	}
	return out, nil
}
