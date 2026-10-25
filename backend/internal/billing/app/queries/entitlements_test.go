package queries_test

import (
	"context"
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/billing"
	"github.com/joaquinBrassinne/MOVO/internal/billing/adapters/configcatalog"
	"github.com/joaquinBrassinne/MOVO/internal/billing/adapters/fixedplan"
	"github.com/joaquinBrassinne/MOVO/internal/billing/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/billing/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
)

type planByBusiness map[int64]domain.Plan

func (p planByBusiness) PlanOf(_ context.Context, businessID int64) (domain.Plan, error) {
	return p[businessID], nil
}

func newCatalog(t *testing.T) *configcatalog.Catalog {
	t.Helper()

	catalog, err := configcatalog.New(config.Plans{
		Starter: config.PlanSettings{MaxBranches: 2, MaxActiveServices: 3, Price: 1},
		Pro:     config.PlanSettings{MaxBranches: 4, MaxActiveServices: 0, Price: 2},
	})
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	return catalog
}

func TestEntitlementsForFollowsThePlanOfTheBusiness(t *testing.T) {
	t.Parallel()

	query := queries.NewEntitlementsFor(newCatalog(t), planByBusiness{1: domain.PlanStarter, 2: domain.PlanPro})

	starter, err := query.Handle(t.Context(), 1)
	if err != nil || starter.CanAddBranch(2) {
		t.Errorf("starter: can add a third branch = %v, err %v; want the limit of 2", starter.CanAddBranch(2), err)
	}
	pro, err := query.Handle(t.Context(), 2)
	if err != nil || !pro.CanAddBranch(3) || !pro.CanActivateService(500) {
		t.Errorf("pro limits wrong, err %v", err)
	}
}

func TestEntitlementsForReportsAnUnknownPlan(t *testing.T) {
	t.Parallel()

	query := queries.NewEntitlementsFor(newCatalog(t), planByBusiness{1: "free"})
	if _, err := query.Handle(t.Context(), 1); !errors.Is(err, billing.ErrUnknownPlan) {
		t.Fatalf("error = %v, want ErrUnknownPlan", err)
	}
}

func TestFixedPlanResolverAnswersTheSamePlanForEveryone(t *testing.T) {
	t.Parallel()

	resolver := fixedplan.New(domain.PlanStarter)
	for _, id := range []int64{1, 99} {
		if got, err := resolver.PlanOf(t.Context(), id); err != nil || got != domain.PlanStarter {
			t.Errorf("PlanOf(%d) = %q, %v", id, got, err)
		}
	}
}
