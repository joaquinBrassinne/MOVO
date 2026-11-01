package queries_test

import (
	"context"
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/queries"
)

type fakeRepeat struct {
	got   benefits.RepeatCustomers
	asked int
}

func (f *fakeRepeat) RepeatCustomers(context.Context, int64, benefits.MetricsPeriod, benefits.MetricsPeriod) (benefits.RepeatCustomers, error) {
	f.asked++
	return f.got, nil
}

func TestRepeatCustomersGivesCountsAndTheShareThatCameBack(t *testing.T) {
	t.Parallel()

	reader := &fakeRepeat{got: benefits.RepeatCustomers{
		Current:          benefits.CustomerCounts{OneRedemption: 15, Repeat: 5},
		Previous:         benefits.CustomerCounts{OneRedemption: 18, Repeat: 2},
		CouponsGenerated: 40, PreviousCouponsGenerated: 30,
	}}
	got, err := queries.NewGetRepeatCustomers(proGate(), reader, clock).Handle(t.Context(), 7, 30)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != queries.StatusOK || got.Customers != 20 || got.PreviousCustomers != 20 || got.OneRedemption != 15 || got.TwoOrMore != 5 {
		t.Fatalf("result = %+v, want 20 customers, 15 once and 5 more", got)
	}
	near(t, "repeat share", got.Repeat.Value, 0.25)
	near(t, "repeat share before", got.Repeat.Previous, 0.1)
	near(t, "change", got.Repeat.Change, 0.15)
}

func TestRepeatCustomersWithNobodyHasNoShareAndNeedsTheSample(t *testing.T) {
	t.Parallel()

	reader := &fakeRepeat{got: benefits.RepeatCustomers{CouponsGenerated: 25, PreviousCouponsGenerated: 25}}
	got, err := queries.NewGetRepeatCustomers(proGate(), reader, clock).Handle(t.Context(), 7, 30)
	if err != nil || got.Status != queries.StatusOK || got.Repeat.Value != nil || got.Customers != 0 {
		t.Errorf("nobody = %+v, %v; want ok with no share to divide", got, err)
	}

	reader.got = benefits.RepeatCustomers{Current: benefits.CustomerCounts{Repeat: 3}, CouponsGenerated: 19}
	got, _ = queries.NewGetRepeatCustomers(proGate(), reader, clock).Handle(t.Context(), 7, 30)
	if got.Status != queries.StatusInsufficientData || got.TwoOrMore != 0 || got.Repeat.Value != nil {
		t.Errorf("19 coupons = %+v, want insufficient_data with no count", got)
	}
}

func TestRepeatCustomersIsRefusedBeforeAnythingIsReadForAStarter(t *testing.T) {
	t.Parallel()

	reader := &fakeRepeat{}
	denied := queries.NewProGate(fakeOwners{business: 42}, &fakePermission{allowed: false})
	if _, err := queries.NewGetRepeatCustomers(denied, reader, clock).Handle(t.Context(), 7, 30); !errors.Is(err, benefits.ErrPlanUpgradeRequired) || reader.asked != 0 {
		t.Errorf("Starter: err = %v, reads = %d; want ErrPlanUpgradeRequired and no read", err, reader.asked)
	}
	if _, err := queries.NewGetRepeatCustomers(proGate(), reader, clock).Handle(t.Context(), 7, 14); !errors.Is(err, benefits.ErrMetricsPeriodInvalid) || reader.asked != 0 {
		t.Errorf("period 14: err = %v, reads = %d; want ErrMetricsPeriodInvalid and no read", err, reader.asked)
	}
}
