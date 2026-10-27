// Package queries holds the read-only benefit use cases.
package queries

import (
	"context"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

// ListMyBenefits returns the benefits of the owner's business for the panel.
type ListMyBenefits struct {
	owners   benefits.OwnerBusinesses
	benefits benefits.BenefitRepository
}

// NewListMyBenefits wires the ListMyBenefits query.
func NewListMyBenefits(owners benefits.OwnerBusinesses, repo benefits.BenefitRepository) *ListMyBenefits {
	return &ListMyBenefits{owners: owners, benefits: repo}
}

// Handle lists every benefit, the inactive ones included, oldest first.
func (q *ListMyBenefits) Handle(ctx context.Context, ownerID int64) ([]domain.Benefit, error) {
	businessID, err := q.owners.EditableBusinessOf(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("resolve business: %w", err)
	}
	list, err := q.benefits.List(ctx, businessID)
	if err != nil {
		return nil, fmt.Errorf("list benefits: %w", err)
	}
	return list, nil
}
