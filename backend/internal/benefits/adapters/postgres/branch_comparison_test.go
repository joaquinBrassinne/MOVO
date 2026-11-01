package postgres_test

import (
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/adapters/postgres"
)

func TestBranchComparisonCountsEachBranchInBothPeriodsAndListsTheRightOnes(t *testing.T) {
	t.Parallel()

	f := newConversionFixture(t, "studio")
	ctx := t.Context()
	var north, retired, quiet int64
	insertBranch := func(dst *int64, name string, active bool) {
		if err := f.pool.QueryRow(ctx, `
INSERT INTO branches (business_id, name, address, city, location, active)
VALUES ($1, $2, 'Calle 2', 'Cordoba', ST_SetSRID(ST_MakePoint(-64.19, -31.41), 4326)::geography, $3) RETURNING id`, f.business, name, active).Scan(dst); err != nil {
			t.Fatal(err)
		}
	}
	insertBranch(&north, "Norte", true)
	insertBranch(&retired, "Cerrada", false)
	insertBranch(&quiet, "Sin actividad", true)
	var closedQuiet int64
	insertBranch(&closedQuiet, "Cerrada sin actividad", false)

	current := periodEnding(time.Date(2028, 3, 15, 0, 0, 0, 0, argentina), 7)
	previous := periodEnding(time.Date(2028, 3, 8, 0, 0, 0, 0, argentina), 7)
	now := time.Date(2028, 3, 12, 10, 0, 0, 0, argentina)
	before := time.Date(2028, 3, 5, 10, 0, 0, 0, argentina)

	var yoga int64
	if err := f.pool.QueryRow(ctx, `INSERT INTO services (business_id, name) VALUES ($1, 'Yoga') RETURNING id`, f.business).Scan(&yoga); err != nil {
		t.Fatal(err)
	}
	f.metric(t, "service_view", "2028-03-12", f.branch, yoga, 20)
	f.metric(t, "service_view", "2028-03-05", f.branch, yoga, 8) // previous period
	f.metric(t, "whatsapp_click", "2028-03-12", f.branch, 0, 6)
	f.metric(t, "whatsapp_click", "2028-03-13", north, 0, 4)
	f.metric(t, "whatsapp_click", "2028-03-06", retired, 0, 2)    // the retired branch only had activity before
	f.metric(t, "service_view", "2028-03-01", f.branch, yoga, 99) // before both periods

	f.coupon(t, now, true, f.branch)
	f.coupon(t, now, true, f.branch)
	f.coupon(t, now, true, north)
	f.coupon(t, before, true, north)
	f.coupon(t, before, true, retired)

	got, err := postgres.NewCoupons(f.pool).BranchComparison(ctx, f.business, current, previous)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]benefits.BranchActivity{}
	for _, b := range got.Branches {
		byName[b.Name] = b
	}
	if len(byName) != 4 {
		t.Fatalf("branches = %v, want Centro, Norte, Cerrada (it had activity) and Sin actividad, not the idle retired one", got.Branches)
	}
	if _, listed := byName["Cerrada sin actividad"]; listed {
		t.Error("a retired branch with no activity must not be listed")
	}
	centro := byName["Centro"]
	if centro.Current != (benefits.BranchCounts{ServiceViews: 20, WhatsAppClicks: 6, Redemptions: 2}) ||
		centro.Previous != (benefits.BranchCounts{ServiceViews: 8}) || !centro.Active {
		t.Errorf("Centro = %+v, want 20 views, 6 taps, 2 redemptions against 8 views", centro)
	}
	if n := byName["Norte"]; n.Current != (benefits.BranchCounts{WhatsAppClicks: 4, Redemptions: 1}) || n.Previous != (benefits.BranchCounts{Redemptions: 1}) {
		t.Errorf("Norte = %+v, want 4 taps and 1 redemption against 1 redemption", n)
	}
	if c := byName["Cerrada"]; c.Active || c.Current != (benefits.BranchCounts{}) || c.Previous != (benefits.BranchCounts{WhatsAppClicks: 2, Redemptions: 1}) {
		t.Errorf("Cerrada = %+v, want it inactive, with only a previous history", c)
	}
	if q := byName["Sin actividad"]; q.Current != (benefits.BranchCounts{}) || q.Previous != (benefits.BranchCounts{}) || !q.Active {
		t.Errorf("Sin actividad = %+v, want zeros", q)
	}
	if got.CouponsGenerated != 3 || got.PreviousCouponsGenerated != 2 {
		t.Errorf("sample = %d now, %d before; want 3 and 2", got.CouponsGenerated, got.PreviousCouponsGenerated)
	}
}

func TestBranchComparisonNeverShowsAnotherBusinessesBranches(t *testing.T) {
	t.Parallel()

	f := newConversionFixture(t, "studio")
	current := periodEnding(time.Date(2028, 3, 15, 0, 0, 0, 0, argentina), 7)
	previous := periodEnding(time.Date(2028, 3, 8, 0, 0, 0, 0, argentina), 7)
	when := time.Date(2028, 3, 12, 10, 0, 0, 0, argentina)

	other := newConversionFixtureIn(t, f, "other")
	other.coupon(t, when, true, other.branch)
	other.metric(t, "whatsapp_click", "2028-03-12", other.branch, 0, 50)

	got, err := postgres.NewCoupons(f.pool).BranchComparison(t.Context(), f.business, current, previous)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Branches) != 1 || got.Branches[0].BranchID != f.branch || got.Branches[0].Current != (benefits.BranchCounts{}) {
		t.Errorf("branches = %+v, want only this business's own branch with no activity", got.Branches)
	}
}
