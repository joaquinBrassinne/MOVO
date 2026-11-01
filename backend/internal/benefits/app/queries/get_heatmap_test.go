package queries_test

import (
	"context"
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/queries"
)

type fakeHeatmap struct {
	got   benefits.Heatmap
	asked int
}

func (f *fakeHeatmap) Heatmap(context.Context, int64, benefits.MetricsPeriod, benefits.MetricsPeriod) (benefits.Heatmap, error) {
	f.asked++
	return f.got, nil
}

func TestHeatmapFillsEveryWeekdayAndSlotInOrderWithZerosForTheEmptyOnes(t *testing.T) {
	t.Parallel()

	reader := &fakeHeatmap{got: benefits.Heatmap{
		CouponsGenerated: 30, Redemptions: 9, PreviousRedemptions: 4,
		// Out of order on purpose: the query orders them.
		Cells: []benefits.HeatCell{{Weekday: 7, Slot: 3, Redemptions: 2}, {Weekday: 1, Slot: 0, Redemptions: 7}},
	}}
	got, err := queries.NewGetHeatmap(proGate(), reader, clock).Handle(t.Context(), 7, 30)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != queries.StatusOK || got.Redemptions != 9 || got.PreviousRedemptions != 4 {
		t.Fatalf("result = %+v, want ok with 9 redemptions against 4", got)
	}
	if len(got.Grid) != 7*len(queries.HeatSlots) {
		t.Fatalf("grid has %d cells, want 28", len(got.Grid))
	}
	for i, c := range got.Grid {
		if c.Weekday != i/4+1 || c.Slot != i%4 {
			t.Fatalf("cell %d = weekday %d slot %d, want weekday %d slot %d", i, c.Weekday, c.Slot, i/4+1, i%4)
		}
	}
	if got.Grid[0].Redemptions != 7 || got.Grid[27].Redemptions != 2 {
		t.Errorf("first %d, last %d; want 7 and 2", got.Grid[0].Redemptions, got.Grid[27].Redemptions)
	}
	var sum int64
	for _, c := range got.Grid {
		sum += c.Redemptions
	}
	if sum != 9 {
		t.Errorf("cells add up to %d, want 9", sum)
	}
}

func TestHeatmapSlotsCoverTheWholeDayInSixHourSteps(t *testing.T) {
	t.Parallel()

	want := 0
	for i, s := range queries.HeatSlots {
		if s.StartHour != want || s.EndHour != want+5 || s.StartHour/6 != i {
			t.Errorf("slot %d = %+v, want hours %d to %d", i, s, want, want+5)
		}
		want += 6
	}
	if want != 24 {
		t.Errorf("slots cover %d hours, want 24", want)
	}
}

func TestHeatmapNeedsTwentyCouponsAndTheProPlan(t *testing.T) {
	t.Parallel()

	reader := &fakeHeatmap{got: benefits.Heatmap{CouponsGenerated: 19, Redemptions: 5, Cells: []benefits.HeatCell{{Weekday: 1, Slot: 1, Redemptions: 5}}}}
	got, err := queries.NewGetHeatmap(proGate(), reader, clock).Handle(t.Context(), 7, 30)
	if err != nil || got.Status != queries.StatusInsufficientData || got.Grid != nil || got.CouponsGenerated != 19 {
		t.Errorf("19 coupons = %+v, %v; want insufficient_data with no grid", got, err)
	}
	reader.got.CouponsGenerated = 20
	if got, _ := queries.NewGetHeatmap(proGate(), reader, clock).Handle(t.Context(), 7, 30); got.Status != queries.StatusOK {
		t.Errorf("20 coupons = %q, want ok", got.Status)
	}

	reader.asked = 0
	denied := queries.NewProGate(fakeOwners{business: 42}, &fakePermission{allowed: false})
	if _, err := queries.NewGetHeatmap(denied, reader, clock).Handle(t.Context(), 7, 30); !errors.Is(err, benefits.ErrPlanUpgradeRequired) || reader.asked != 0 {
		t.Errorf("Starter: err = %v, reads = %d; want ErrPlanUpgradeRequired and no read", err, reader.asked)
	}
	if _, err := queries.NewGetHeatmap(proGate(), reader, clock).Handle(t.Context(), 7, 14); !errors.Is(err, benefits.ErrMetricsPeriodInvalid) || reader.asked != 0 {
		t.Errorf("period 14: err = %v, reads = %d; want ErrMetricsPeriodInvalid and no read", err, reader.asked)
	}
}
