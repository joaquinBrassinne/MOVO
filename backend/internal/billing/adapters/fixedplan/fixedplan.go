// Package fixedplan is the interim billing.PlanResolver: it puts every business
// on one plan until subscriptions exist (phase 5).
package fixedplan

import (
	"context"

	"github.com/joaquinBrassinne/MOVO/internal/billing/domain"
)

// Resolver answers the same plan for every business.
type Resolver struct {
	plan domain.Plan
}

// New builds a resolver that always answers plan.
func New(plan domain.Plan) *Resolver { return &Resolver{plan: plan} }

// PlanOf returns the fixed plan.
func (r *Resolver) PlanOf(context.Context, int64) (domain.Plan, error) { return r.plan, nil }
