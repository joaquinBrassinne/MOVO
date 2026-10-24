// Package billing is the module that knows the plans and what each one allows.
// This file declares the ports the application layer consumes; adapters
// implement them.
package billing

import (
	"errors"

	"github.com/joaquinBrassinne/MOVO/internal/billing/domain"
)

// ErrUnknownPlan means the catalog has no entry for the plan.
var ErrUnknownPlan = errors.New("billing: plan is not in the catalog")

// PlanInfo is a plan as shown to owners.
type PlanInfo struct {
	Plan         domain.Plan
	Price        int
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
