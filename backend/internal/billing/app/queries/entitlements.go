// Package queries holds the read use cases of the billing module.
package queries

import (
	"context"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/billing"
	"github.com/joaquinBrassinne/MOVO/internal/billing/domain"
)

// EntitlementsFor resolves what a business may have from its plan.
type EntitlementsFor struct {
	catalog billing.PlanCatalog
	plans   billing.PlanResolver
}

// NewEntitlementsFor wires the EntitlementsFor query.
func NewEntitlementsFor(catalog billing.PlanCatalog, plans billing.PlanResolver) *EntitlementsFor {
	return &EntitlementsFor{catalog: catalog, plans: plans}
}

// Handle returns the entitlements of the business, or billing.ErrUnknownPlan when
// its plan is not in the catalog.
func (q *EntitlementsFor) Handle(ctx context.Context, businessID int64) (domain.Entitlements, error) {
	plan, err := q.plans.PlanOf(ctx, businessID)
	if err != nil {
		return domain.Entitlements{}, fmt.Errorf("resolve plan: %w", err)
	}
	entitlements, err := q.catalog.Entitlements(plan)
	if err != nil {
		return domain.Entitlements{}, fmt.Errorf("plan %q: %w", plan, err)
	}
	return entitlements, nil
}
