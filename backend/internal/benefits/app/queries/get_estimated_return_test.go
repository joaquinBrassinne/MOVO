package queries_test

import (
	"context"
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/queries"
)

type fakeEconomics struct{ econ benefits.Economics }

func (f fakeEconomics) Economics(context.Context, int64) (benefits.Economics, error) {
	return f.econ, nil
}

type fakeTotals struct {
	got   benefits.RedemptionTotals
	asked int
}

func (f *fakeTotals) RedemptionTotals(context.Context, int64, benefits.MetricsPeriod, benefits.MetricsPeriod) (benefits.RedemptionTotals, error) {
	f.asked++
	return f.got, nil
}

func econ(ticket int) fakeEconomics {
	return fakeEconomics{econ: benefits.Economics{AverageTicketARS: &ticket, PlanPriceARS: 29800}}
}

func TestEstimatedReturnMultipliesTheRedemptionsByTheTicketAndSetsThePlanBeside(t *testing.T) {
	t.Parallel()

	reader := &fakeTotals{got: benefits.RedemptionTotals{
		Redemptions: 22, PreviousRedemptions: 10, CouponsGenerated: 30, PreviousCouponsGenerated: 25,
	}}
	got, err := queries.NewGetEstimatedReturn(proGate(), econ(15000), reader, clock).Handle(t.Context(), 7, 30)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != queries.StatusOK || got.EstimatedARS != 330000 || got.AverageTicketARS != 15000 || got.PlanPriceARS != 29800 {
		t.Fatalf("result = %+v, want 22 x 15000 = 330000 beside a plan of 29800", got)
	}
	if got.PreviousEstimatedARS == nil || *got.PreviousEstimatedARS != 150000 || got.Redemptions != 22 || got.PreviousRedemptions != 10 {
		t.Errorf("previous = %v with %d/%d redemptions, want 150000 from 10", got.PreviousEstimatedARS, got.Redemptions, got.PreviousRedemptions)
	}
}

func TestEstimatedReturnIsNotShownWithoutATicketAndReadsNothing(t *testing.T) {
	t.Parallel()

	reader := &fakeTotals{got: benefits.RedemptionTotals{Redemptions: 50, CouponsGenerated: 100}}
	none := fakeEconomics{econ: benefits.Economics{PlanPriceARS: 29800}}
	got, err := queries.NewGetEstimatedReturn(proGate(), none, reader, clock).Handle(t.Context(), 7, 30)
	if err != nil || got.Status != queries.StatusNoTicket || got.EstimatedARS != 0 || got.PlanPriceARS != 0 || reader.asked != 0 {
		t.Errorf("no ticket = %+v, %v (reads %d); want no_ticket with no figure and no read", got, err, reader.asked)
	}
}

func TestEstimatedReturnNeedsTwentyCouponsAndNoBaselineBelowThat(t *testing.T) {
	t.Parallel()

	reader := &fakeTotals{got: benefits.RedemptionTotals{Redemptions: 5, CouponsGenerated: 19}}
	got, err := queries.NewGetEstimatedReturn(proGate(), econ(15000), reader, clock).Handle(t.Context(), 7, 30)
	if err != nil || got.Status != queries.StatusInsufficientData || got.EstimatedARS != 0 || got.CouponsGenerated != 19 {
		t.Errorf("19 coupons = %+v, %v; want insufficient_data and no figure", got, err)
	}

	reader.got = benefits.RedemptionTotals{Redemptions: 5, PreviousRedemptions: 9, CouponsGenerated: 20, PreviousCouponsGenerated: 19}
	got, _ = queries.NewGetEstimatedReturn(proGate(), econ(15000), reader, clock).Handle(t.Context(), 7, 30)
	if got.Status != queries.StatusOK || got.EstimatedARS != 75000 || got.PreviousEstimatedARS != nil {
		t.Errorf("20 coupons = %+v, want 75000 and no baseline from a previous period of 19", got)
	}
}

func TestEstimatedReturnDoesNotOverflowWithBigNumbers(t *testing.T) {
	t.Parallel()

	reader := &fakeTotals{got: benefits.RedemptionTotals{Redemptions: 100000, CouponsGenerated: 200000}}
	got, err := queries.NewGetEstimatedReturn(proGate(), econ(2147483647), reader, clock).Handle(t.Context(), 7, 30)
	if err != nil || got.EstimatedARS != 100000*2147483647 {
		t.Errorf("estimate = %d, %v; want %d", got.EstimatedARS, err, int64(100000)*2147483647)
	}
}

func TestEstimatedReturnIsRefusedBeforeAnythingIsReadForAStarter(t *testing.T) {
	t.Parallel()

	reader := &fakeTotals{}
	denied := queries.NewProGate(fakeOwners{business: 42}, &fakePermission{allowed: false})
	if _, err := queries.NewGetEstimatedReturn(denied, econ(15000), reader, clock).Handle(t.Context(), 7, 30); !errors.Is(err, benefits.ErrPlanUpgradeRequired) || reader.asked != 0 {
		t.Errorf("Starter: err = %v, reads = %d; want ErrPlanUpgradeRequired and no read", err, reader.asked)
	}
	if _, err := queries.NewGetEstimatedReturn(proGate(), econ(15000), reader, clock).Handle(t.Context(), 7, 14); !errors.Is(err, benefits.ErrMetricsPeriodInvalid) || reader.asked != 0 {
		t.Errorf("period 14: err = %v, reads = %d; want ErrMetricsPeriodInvalid and no read", err, reader.asked)
	}
}
