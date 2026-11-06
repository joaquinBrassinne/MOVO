package configcatalog_test

import (
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/billing"
	"github.com/joaquinBrassinne/MOVO/internal/billing/adapters/configcatalog"
	"github.com/joaquinBrassinne/MOVO/internal/billing/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
)

var plans = config.Plans{
	Starter: config.PlanSettings{MaxBranches: 2, MaxActiveServices: 3, Price: 19800},
	Pro:     config.PlanSettings{MaxBranches: 4, MaxActiveServices: 0, Price: 29800},
}

func TestCatalogReadsLimitsFromConfiguration(t *testing.T) {
	t.Parallel()

	catalog, err := configcatalog.New(plans)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	starter, err := catalog.Entitlements(domain.PlanStarter)
	if err != nil || starter.MaxBranches() != 2 || starter.MaxActiveServices() != 3 {
		t.Errorf("starter = %+v, %v", starter, err)
	}
	pro, err := catalog.Entitlements(domain.PlanPro)
	if err != nil || pro.MaxBranches() != 4 || pro.MaxActiveServices() != 0 {
		t.Errorf("pro = %+v, %v", pro, err)
	}

	if _, err := catalog.Entitlements("free"); !errors.Is(err, billing.ErrUnknownPlan) {
		t.Errorf("unknown plan error = %v, want ErrUnknownPlan", err)
	}
}

func TestCatalogGivesTheAdvancedMetricsOnlyToThePlansConfiguredForThem(t *testing.T) {
	t.Parallel()

	catalog, err := configcatalog.New(config.Plans{
		Starter: config.PlanSettings{MaxBranches: 2, MaxActiveServices: 3, Price: 19800},
		Pro:     config.PlanSettings{MaxBranches: 4, MaxActiveServices: 0, Price: 29800, AdvancedMetrics: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	starter, _ := catalog.Entitlements(domain.PlanStarter)
	pro, _ := catalog.Entitlements(domain.PlanPro)
	if starter.AdvancedMetrics() || !pro.AdvancedMetrics() {
		t.Errorf("starter %t, pro %t; want false and true", starter.AdvancedMetrics(), pro.AdvancedMetrics())
	}
	for _, info := range catalog.Plans() {
		if info.Plan == domain.PlanPro && !info.Entitlements.AdvancedMetrics() {
			t.Error("the plan list must carry the same capability")
		}
	}
}

func TestCatalogListsPlansCheapestFirstWithPrices(t *testing.T) {
	t.Parallel()

	catalog, err := configcatalog.New(plans)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	got := catalog.Plans()
	if len(got) != 2 || got[0].Plan != domain.PlanStarter || got[0].Price != 19800 ||
		got[1].Plan != domain.PlanPro || got[1].Price != 29800 {
		t.Fatalf("plans = %+v", got)
	}

	got[0].Price = 1
	if catalog.Plans()[0].Price != 19800 {
		t.Error("callers must not be able to change the catalog through the returned slice")
	}
}

func TestCatalogRejectsInvalidLimits(t *testing.T) {
	t.Parallel()

	bad := plans
	bad.Starter.MaxBranches = -1
	if _, err := configcatalog.New(bad); !errors.Is(err, domain.ErrLimitInvalid) {
		t.Fatalf("error = %v, want ErrLimitInvalid", err)
	}
}
