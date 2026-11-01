package queries_test

import (
	"context"
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/queries"
)

type fakeBranches struct {
	got   benefits.BranchComparison
	asked int
}

func (f *fakeBranches) BranchComparison(context.Context, int64, benefits.MetricsPeriod, benefits.MetricsPeriod) (benefits.BranchComparison, error) {
	f.asked++
	return f.got, nil
}

func TestBranchComparisonGrowsAgainstThePreviousPeriodAndRanksByRedemptions(t *testing.T) {
	t.Parallel()

	reader := &fakeBranches{got: benefits.BranchComparison{
		CouponsGenerated: 40, PreviousCouponsGenerated: 30,
		Branches: []benefits.BranchActivity{
			{BranchID: 1, Name: "Centro", Active: true,
				Current:  benefits.BranchCounts{ServiceViews: 30, WhatsAppClicks: 10, Redemptions: 5},
				Previous: benefits.BranchCounts{ServiceViews: 20, WhatsAppClicks: 0, Redemptions: 10}},
			{BranchID: 2, Name: "Norte", Active: true,
				Current:  benefits.BranchCounts{Redemptions: 15},
				Previous: benefits.BranchCounts{Redemptions: 12}},
		},
	}}
	got, err := queries.NewGetBranchComparison(proGate(), reader, clock).Handle(t.Context(), 7, 30)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != queries.StatusOK || len(got.Branches) != 2 || got.Branches[0].Name != "Norte" {
		t.Fatalf("result = %+v, want two branches with Norte (more redemptions) first", got)
	}
	near(t, "norte redemptions growth", got.Branches[0].Redemptions.Change, 0.25)
	centro := got.Branches[1]
	near(t, "centro views growth", centro.ServiceViews.Change, 0.5)
	near(t, "centro redemptions growth", centro.Redemptions.Change, -0.5)
	if centro.WhatsAppClicks.Change != nil || centro.WhatsAppClicks.Previous != 0 {
		t.Errorf("clicks from zero = %+v, want no growth to compute", centro.WhatsAppClicks)
	}
	if centro.ServiceViews.Value != 30 || centro.ServiceViews.Previous != 20 {
		t.Errorf("views = %+v, want 30 against 20", centro.ServiceViews)
	}
}

func TestBranchComparisonHasNoGrowthFromAPreviousPeriodWithTooFewCoupons(t *testing.T) {
	t.Parallel()

	reader := &fakeBranches{got: benefits.BranchComparison{
		CouponsGenerated: 25, PreviousCouponsGenerated: 19,
		Branches: []benefits.BranchActivity{{BranchID: 1, Name: "Centro", Active: true,
			Current: benefits.BranchCounts{Redemptions: 8}, Previous: benefits.BranchCounts{Redemptions: 4}}},
	}}
	got, err := queries.NewGetBranchComparison(proGate(), reader, clock).Handle(t.Context(), 7, 30)
	if err != nil || got.Status != queries.StatusOK {
		t.Fatalf("result = %+v, %v; want ok", got, err)
	}
	if r := got.Branches[0].Redemptions; r.Value != 8 || r.Previous != 4 || r.Change != nil {
		t.Errorf("redemptions = %+v, want the counts and no growth from 19 coupons", r)
	}
}

func TestBranchComparisonNeedsTwentyCouponsAndTheProPlan(t *testing.T) {
	t.Parallel()

	reader := &fakeBranches{got: benefits.BranchComparison{CouponsGenerated: 19, Branches: []benefits.BranchActivity{{BranchID: 1, Name: "Centro"}}}}
	got, err := queries.NewGetBranchComparison(proGate(), reader, clock).Handle(t.Context(), 7, 30)
	if err != nil || got.Status != queries.StatusInsufficientData || got.Branches != nil || got.CouponsGenerated != 19 {
		t.Errorf("19 coupons = %+v, %v; want insufficient_data with no branches", got, err)
	}

	reader.asked = 0
	denied := queries.NewProGate(fakeOwners{business: 42}, &fakePermission{allowed: false})
	if _, err := queries.NewGetBranchComparison(denied, reader, clock).Handle(t.Context(), 7, 30); !errors.Is(err, benefits.ErrPlanUpgradeRequired) || reader.asked != 0 {
		t.Errorf("Starter: err = %v, reads = %d; want ErrPlanUpgradeRequired and no read", err, reader.asked)
	}
	if _, err := queries.NewGetBranchComparison(proGate(), reader, clock).Handle(t.Context(), 7, 14); !errors.Is(err, benefits.ErrMetricsPeriodInvalid) || reader.asked != 0 {
		t.Errorf("period 14: err = %v, reads = %d; want ErrMetricsPeriodInvalid and no read", err, reader.asked)
	}
}
