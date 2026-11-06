// Package pilotplan is a TEMPORARY billing.PlanResolver: pilot businesses get the
// Pro limits for free during the pilot. When the pilot ends, delete this package
// and its wiring in main.go; those businesses then pay for Pro like everyone else.
package pilotplan

import (
	"context"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/billing"
	"github.com/joaquinBrassinne/MOVO/internal/billing/domain"
)

// PilotLookup tells whether a business is part of the pilot. The business module
// owns that flag; billing only asks.
type PilotLookup interface {
	IsPilot(ctx context.Context, businessID int64) (bool, error)
}

// Resolver puts pilots on Pro and defers to another resolver for the rest.
type Resolver struct {
	pilots   PilotLookup
	fallback billing.PlanResolver
}

// New builds the resolver. fallback answers for every non-pilot business.
func New(pilots PilotLookup, fallback billing.PlanResolver) *Resolver {
	return &Resolver{pilots: pilots, fallback: fallback}
}

// PlanOf returns Pro for pilots and the fallback's plan for everyone else. A
// failed lookup is an error, never a guess.
func (r *Resolver) PlanOf(ctx context.Context, businessID int64) (domain.Plan, error) {
	pilot, err := r.pilots.IsPilot(ctx, businessID)
	if err != nil {
		return "", fmt.Errorf("pilotplan: look up pilot flag: %w", err)
	}
	if pilot {
		return domain.PlanPro, nil
	}
	return r.fallback.PlanOf(ctx, businessID)
}
