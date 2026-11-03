package queries_test

import (
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/billing"
	"github.com/joaquinBrassinne/MOVO/internal/billing/adapters/configcatalog"
	"github.com/joaquinBrassinne/MOVO/internal/billing/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/billing/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
)

func TestListSubscriptionsMarksTheBusinessesWithPrioritySupport(t *testing.T) {
	t.Parallel()

	catalog, err := configcatalog.New(config.Plans{
		Starter: config.PlanSettings{MaxBranches: 2, MaxActiveServices: 3, Price: 1},
		Pro:     config.PlanSettings{MaxBranches: 4, MaxActiveServices: 0, Price: 2, PrioritySupport: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	starter := row(t, 1, "Starter", "", "2026-12-01")
	pro := row(t, 2, "Pro", "", "2026-12-02")
	pro.Subscription.Plan = domain.PlanPro
	pilot := row(t, 3, "Pilot", "", "2026-12-03") // a pilot is on its subscription's plan, but gets Pro's support
	pilot.Pilot = true

	got, err := queries.NewListSubscriptions(directory{rows: []billing.SubscriptionRow{starter, pro, pilot}}).
		WithPlans(catalog).Handle(t.Context(), queries.SubscriptionFilter{}, day(t, "2026-11-12"))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"Starter": false, "Pro": true, "Pilot": true}
	for _, e := range got.Entries {
		if e.PrioritySupport != want[e.BusinessName] {
			t.Errorf("%s: priority support = %t, want %t", e.BusinessName, e.PrioritySupport, want[e.BusinessName])
		}
	}
}

func TestListSubscriptionsWithoutACatalogMarksNobody(t *testing.T) {
	t.Parallel()

	got, err := queries.NewListSubscriptions(directory{rows: []billing.SubscriptionRow{row(t, 1, "A", "", "2026-12-01")}}).
		Handle(t.Context(), queries.SubscriptionFilter{}, day(t, "2026-11-12"))
	if err != nil || len(got.Entries) != 1 || got.Entries[0].PrioritySupport {
		t.Errorf("entries = %+v, %v; want one without the mark", got.Entries, err)
	}
}
