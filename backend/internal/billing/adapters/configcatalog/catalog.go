// Package configcatalog implements billing.PlanCatalog from the environment
// configuration (SDD §4.2 and §7.5).
package configcatalog

import (
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/billing"
	"github.com/joaquinBrassinne/MOVO/internal/billing/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
)

// Catalog implements billing.PlanCatalog.
type Catalog struct {
	plans []billing.PlanInfo
}

// New builds the catalog from the loaded settings.
func New(settings config.Plans) (*Catalog, error) {
	starter, err := info(domain.PlanStarter, settings.Starter)
	if err != nil {
		return nil, err
	}
	pro, err := info(domain.PlanPro, settings.Pro)
	if err != nil {
		return nil, err
	}
	return &Catalog{plans: []billing.PlanInfo{starter, pro}}, nil
}

func info(plan domain.Plan, s config.PlanSettings) (billing.PlanInfo, error) {
	entitlements, err := domain.NewEntitlements(s.MaxBranches, s.MaxActiveServices)
	if err != nil {
		return billing.PlanInfo{}, fmt.Errorf("configcatalog: plan %s: %w", plan, err)
	}
	return billing.PlanInfo{Plan: plan, Price: s.Price, Entitlements: entitlements}, nil
}

// Entitlements returns the limits of the plan.
func (c *Catalog) Entitlements(plan domain.Plan) (domain.Entitlements, error) {
	for _, p := range c.plans {
		if p.Plan == plan {
			return p.Entitlements, nil
		}
	}
	return domain.Entitlements{}, billing.ErrUnknownPlan
}

// Plans lists every plan, cheapest first. The slice is a copy.
func (c *Catalog) Plans() []billing.PlanInfo {
	return append([]billing.PlanInfo(nil), c.plans...)
}
