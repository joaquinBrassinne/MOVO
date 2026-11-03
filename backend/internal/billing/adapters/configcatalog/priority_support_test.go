package configcatalog_test

import (
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/billing/adapters/configcatalog"
	"github.com/joaquinBrassinne/MOVO/internal/billing/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
)

func TestCatalogGivesThePrioritySupportOnlyToThePlansConfiguredForIt(t *testing.T) {
	t.Parallel()

	catalog, err := configcatalog.New(config.Plans{
		Starter: config.PlanSettings{MaxBranches: 2, MaxActiveServices: 3, Price: 19800},
		Pro:     config.PlanSettings{MaxBranches: 4, MaxActiveServices: 0, Price: 29800, PrioritySupport: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	starter, _ := catalog.Entitlements(domain.PlanStarter)
	pro, _ := catalog.Entitlements(domain.PlanPro)
	if starter.PrioritySupport() || !pro.PrioritySupport() {
		t.Errorf("starter %t, pro %t; want false and true", starter.PrioritySupport(), pro.PrioritySupport())
	}
}
