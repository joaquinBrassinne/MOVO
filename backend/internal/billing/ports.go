// Package billing is the module that knows the plans and what each one allows.
// This file declares the ports the application layer consumes; adapters
// implement them.
package billing

import (
	"context"
	"errors"

	"github.com/joaquinBrassinne/MOVO/internal/billing/domain"
)

// ErrUnknownPlan means the catalog has no entry for the plan.
var ErrUnknownPlan = errors.New("billing: plan is not in the catalog")

// PlanInfo is a plan as shown to owners.
type PlanInfo struct {
	Plan  domain.Plan
	Price int
	// CheckoutURL is where the owner pays: the Mercado Pago link of the plan.
	CheckoutURL  string
	Entitlements domain.Entitlements
}

// PlanCatalog tells what each plan offers. Today the values come from
// configuration; a database-backed catalog can replace it without touching the
// domain.
type PlanCatalog interface {
	// Entitlements returns the limits of the plan or ErrUnknownPlan.
	Entitlements(plan domain.Plan) (domain.Entitlements, error)
	// Plans lists every plan, cheapest first.
	Plans() []PlanInfo
}

// PlanResolver tells which plan a business is on. Subscriptions arrive in
// phase 5; until then a fixed resolver answers for everyone.
type PlanResolver interface {
	PlanOf(ctx context.Context, businessID int64) (domain.Plan, error)
}

// ErrOwnerHasNoBusiness means the owner has not created a business, so there is no plan.
var ErrOwnerHasNoBusiness = errors.New("billing: owner has no business")

// OwnerPlans tells which plan an owner's business is on. The composition root builds
// it from the business module and the PlanResolver, so billing never reads businesses.
type OwnerPlans interface {
	// PlanOfOwner returns ErrOwnerHasNoBusiness when the owner has no business.
	PlanOfOwner(ctx context.Context, ownerID int64) (PlanInfo, error)
}
