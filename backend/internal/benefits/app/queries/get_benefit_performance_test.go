package queries_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

type fakePerformance struct {
	got   benefits.Performance
	asked int
	at    time.Time
}

func (f *fakePerformance) BenefitPerformance(_ context.Context, _ int64, _, _ benefits.MetricsPeriod, now time.Time) (benefits.Performance, error) {
	f.asked++
	f.at = now
	return f.got, nil
}

func ownedBenefits() fakeBenefits {
	return fakeBenefits{list: []domain.Benefit{
		{ID: 1, Type: domain.TypePercent, Percent: 15},
		{ID: 2, Type: domain.TypePercent, Percent: 10, Title: "Clase de prueba"},
	}}
}

func TestBenefitPerformanceNamesEachBenefitAndRanksByCouponsGenerated(t *testing.T) {
	t.Parallel()

	reader := &fakePerformance{got: benefits.Performance{
		Current: []benefits.BenefitCounts{
			{BenefitID: 1, Generated: 10, Redeemed: 4, Expired: 3},
			{BenefitID: 2, Generated: 30, Redeemed: 15, Expired: 5},
		},
		Previous: []benefits.BenefitCounts{
			{BenefitID: 1, Generated: 10, Redeemed: 2, Expired: 8},
			{BenefitID: 2, Generated: 20, Redeemed: 10, Expired: 5},
		},
	}}
	got, err := queries.NewGetBenefitPerformance(proGate(), ownedBenefits(), reader, clock).Handle(t.Context(), 7, 30)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != queries.StatusOK || got.CouponsGenerated != 40 || len(got.Benefits) != 2 {
		t.Fatalf("result = %+v, want ok with 40 coupons and two benefits", got)
	}
	first, second := got.Benefits[0], got.Benefits[1]
	if first.BenefitID != 2 || first.Name != "Clase de prueba" || second.BenefitID != 1 || second.Name != "-15%" {
		t.Errorf("order/names = %d %q, %d %q; want the titled one first and the other by its label", first.BenefitID, first.Name, second.BenefitID, second.Name)
	}
	if first.Generated != 30 || first.Redeemed != 15 || first.Expired != 5 || first.Pending != 10 {
		t.Errorf("first = %+v, want 30 generated, 15 redeemed, 5 expired, 10 pending", first)
	}
	near(t, "rate", first.Rate, 0.5)
	near(t, "previous rate", first.Previous, 0.5)
	near(t, "change", first.Change, 0)
	near(t, "second rate", second.Rate, 0.4)
	near(t, "second change", second.Change, 0.2)
	if !reader.at.Equal(now) {
		t.Errorf("expiry judged at %v, want %v", reader.at, now)
	}
}

func TestBenefitPerformanceNeedsTwentyCouponsAndNoBaselineBelowThat(t *testing.T) {
	t.Parallel()

	reader := &fakePerformance{got: benefits.Performance{Current: []benefits.BenefitCounts{{BenefitID: 1, Generated: 19, Redeemed: 5}}}}
	got, err := queries.NewGetBenefitPerformance(proGate(), ownedBenefits(), reader, clock).Handle(t.Context(), 7, 30)
	if err != nil || got.Status != queries.StatusInsufficientData || got.Benefits != nil || got.CouponsGenerated != 19 {
		t.Errorf("19 coupons = %+v, %v; want insufficient_data with no benefits", got, err)
	}

	// 20 in all across two benefits is enough; a previous period with 19 is not a baseline.
	reader.got = benefits.Performance{
		Current:  []benefits.BenefitCounts{{BenefitID: 1, Generated: 12, Redeemed: 6}, {BenefitID: 2, Generated: 8, Redeemed: 2}},
		Previous: []benefits.BenefitCounts{{BenefitID: 1, Generated: 19, Redeemed: 19}},
	}
	got, err = queries.NewGetBenefitPerformance(proGate(), ownedBenefits(), reader, clock).Handle(t.Context(), 7, 30)
	if err != nil || got.Status != queries.StatusOK {
		t.Fatalf("20 coupons = %+v, %v; want ok", got, err)
	}
	for _, b := range got.Benefits {
		if b.Previous != nil || b.Change != nil {
			t.Errorf("benefit %d has a baseline from a previous period of 19 coupons: %+v", b.BenefitID, b)
		}
	}
}

func TestBenefitPerformanceIsRefusedBeforeAnythingIsReadForAStarter(t *testing.T) {
	t.Parallel()

	reader := &fakePerformance{}
	denied := queries.NewProGate(fakeOwners{business: 42}, &fakePermission{allowed: false})
	if _, err := queries.NewGetBenefitPerformance(denied, ownedBenefits(), reader, clock).Handle(t.Context(), 7, 30); !errors.Is(err, benefits.ErrPlanUpgradeRequired) || reader.asked != 0 {
		t.Errorf("Starter: err = %v, reads = %d; want ErrPlanUpgradeRequired and no read", err, reader.asked)
	}
	if _, err := queries.NewGetBenefitPerformance(proGate(), ownedBenefits(), reader, clock).Handle(t.Context(), 7, 14); !errors.Is(err, benefits.ErrMetricsPeriodInvalid) || reader.asked != 0 {
		t.Errorf("period 14: err = %v, reads = %d; want ErrMetricsPeriodInvalid and no read", err, reader.asked)
	}
}
