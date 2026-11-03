package queries_test

import (
	"context"
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/queries"
)

type fakeConversion struct {
	got     benefits.Conversion
	asked   int
	current benefits.MetricsPeriod
	prev    benefits.MetricsPeriod
}

func (f *fakeConversion) Conversion(_ context.Context, _ int64, current, previous benefits.MetricsPeriod) (benefits.Conversion, error) {
	f.asked++
	f.current, f.prev = current, previous
	return f.got, nil
}

func proGate() queries.ProGate {
	return queries.NewProGate(fakeOwners{business: 42}, &fakePermission{allowed: true})
}

func counts(views, generated, redeemed int64) benefits.ConversionCounts {
	return benefits.ConversionCounts{ProfileViews: views, CouponsGenerated: generated, CouponsRedeemed: redeemed}
}

func near(t *testing.T, name string, got *float64, want float64) {
	t.Helper()
	if got == nil || *got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func TestConversionRatesCompareWithThePreviousPeriod(t *testing.T) {
	t.Parallel()

	reader := &fakeConversion{got: benefits.Conversion{
		Current:  counts(200, 40, 10),
		Previous: counts(100, 30, 6),
	}}
	got, err := queries.NewGetConversion(proGate(), reader, clock).Handle(t.Context(), 7, 30)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != queries.StatusOK {
		t.Fatalf("status = %q, want ok", got.Status)
	}
	near(t, "views to coupons", got.ViewsToCoupons.Value, 0.2)
	near(t, "views to coupons before", got.ViewsToCoupons.Previous, 0.3)
	near(t, "views to coupons change", got.ViewsToCoupons.Change, -0.1)
	near(t, "coupons to redemptions", got.CouponsToRedemptions.Value, 0.25)
	near(t, "coupons to redemptions before", got.CouponsToRedemptions.Previous, 0.2)
	near(t, "coupons to redemptions change", got.CouponsToRedemptions.Change, 0.05)
	if got.ViewsToCoupons.Numerator != 40 || got.ViewsToCoupons.Denominator != 200 {
		t.Errorf("views to coupons counts = %d/%d, want 40/200", got.ViewsToCoupons.Numerator, got.ViewsToCoupons.Denominator)
	}
}

func TestConversionAsksForThePeriodAndTheOneBeforeItOfTheSameLength(t *testing.T) {
	t.Parallel()

	reader := &fakeConversion{}
	if _, err := queries.NewGetConversion(proGate(), reader, clock).Handle(t.Context(), 7, 7); err != nil {
		t.Fatal(err)
	}
	// "now" is March 15th 2028 in Argentina.
	if reader.current.FromDay != "2028-03-09" || reader.current.ToDay != "2028-03-15" {
		t.Errorf("current = %s..%s, want 2028-03-09..2028-03-15", reader.current.FromDay, reader.current.ToDay)
	}
	if reader.prev.FromDay != "2028-03-02" || reader.prev.ToDay != "2028-03-08" {
		t.Errorf("previous = %s..%s, want 2028-03-02..2028-03-08", reader.prev.FromDay, reader.prev.ToDay)
	}
	if !reader.prev.To.Equal(reader.current.From) || reader.prev.Days != 7 {
		t.Errorf("the previous period must end where the current begins: %v vs %v", reader.prev.To, reader.current.From)
	}

	// Ninety days across a leap February.
	if _, err := queries.NewGetConversion(proGate(), reader, clock).Handle(t.Context(), 7, 90); err != nil {
		t.Fatal(err)
	}
	if reader.current.FromDay != "2027-12-17" || reader.prev.FromDay != "2027-09-18" || reader.prev.ToDay != "2027-12-16" {
		t.Errorf("90 days: current from %s, previous %s..%s", reader.current.FromDay, reader.prev.FromDay, reader.prev.ToDay)
	}
}

func TestConversionNeedsTwentyCouponsBeforeItShowsAnyPercentage(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		generated int64
		want      string
	}{{0, queries.StatusInsufficientData}, {19, queries.StatusInsufficientData}, {20, queries.StatusOK}, {21, queries.StatusOK}} {
		reader := &fakeConversion{got: benefits.Conversion{
			Current:    counts(100, tc.generated, 5),
			ByBranch:   []benefits.BranchConversion{{BranchID: 1, Name: "Centro", WhatsAppClicks: 4, Redemptions: 2}},
			ByActivity: []benefits.ActivityConversion{{ServiceID: 1, Name: "Yoga", Views: 4, WhatsAppClicks: 2}},
		}}
		got, err := queries.NewGetConversion(proGate(), reader, clock).Handle(t.Context(), 7, 30)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != tc.want || got.CouponsGenerated != tc.generated || got.Minimum != 20 {
			t.Errorf("%d coupons: status %q (generated %d, minimum %d), want %q", tc.generated, got.Status, got.CouponsGenerated, got.Minimum, tc.want)
		}
		if tc.want == queries.StatusInsufficientData && (got.ViewsToCoupons.Value != nil || got.CouponsToRedemptions.Value != nil || got.ByBranch != nil || got.ByActivity != nil) {
			t.Errorf("%d coupons: %+v, want no rate and no breakdown", tc.generated, got)
		}
	}
}

func TestConversionDoesNotCompareWithAPreviousPeriodWithTooLittleData(t *testing.T) {
	t.Parallel()

	reader := &fakeConversion{got: benefits.Conversion{Current: counts(200, 40, 10), Previous: counts(50, 19, 9)}}
	got, err := queries.NewGetConversion(proGate(), reader, clock).Handle(t.Context(), 7, 30)
	if err != nil {
		t.Fatal(err)
	}
	near(t, "rate", got.CouponsToRedemptions.Value, 0.25)
	if got.CouponsToRedemptions.Previous != nil || got.CouponsToRedemptions.Change != nil || got.ViewsToCoupons.Change != nil {
		t.Errorf("previous %v, change %v; want no baseline from 19 coupons", got.CouponsToRedemptions.Previous, got.CouponsToRedemptions.Change)
	}
}

func TestConversionHasNoRateWithNothingToDivideBy(t *testing.T) {
	t.Parallel()

	reader := &fakeConversion{got: benefits.Conversion{
		Current:    counts(0, 25, 5),
		Previous:   counts(0, 25, 5),
		ByBranch:   []benefits.BranchConversion{{BranchID: 1, Name: "Centro", WhatsAppClicks: 0, Redemptions: 3}, {BranchID: 2, Name: "Norte", WhatsAppClicks: 10, Redemptions: 4}},
		ByActivity: []benefits.ActivityConversion{{ServiceID: 1, Name: "Yoga", Views: 0, WhatsAppClicks: 2}, {ServiceID: 2, Name: "Pilates", Views: 8, WhatsAppClicks: 2}},
	}}
	got, err := queries.NewGetConversion(proGate(), reader, clock).Handle(t.Context(), 7, 30)
	if err != nil {
		t.Fatal(err)
	}
	if got.ViewsToCoupons.Value != nil || got.ViewsToCoupons.Change != nil {
		t.Errorf("views to coupons with no views = %+v, want no rate", got.ViewsToCoupons)
	}
	if got.ByBranch[0].Rate != nil || got.ByActivity[0].Rate != nil {
		t.Error("a branch with no taps and an activity with no views must have no rate")
	}
	near(t, "branch rate", got.ByBranch[1].Rate, 0.4)
	near(t, "activity rate", got.ByActivity[1].Rate, 0.25)
}

func TestConversionIsRefusedBeforeAnythingIsReadWhenThePlanLacksIt(t *testing.T) {
	t.Parallel()

	reader := &fakeConversion{}
	gate := queries.NewProGate(fakeOwners{business: 42}, &fakePermission{allowed: false})
	_, err := queries.NewGetConversion(gate, reader, clock).Handle(t.Context(), 7, 30)
	if !errors.Is(err, benefits.ErrPlanUpgradeRequired) || reader.asked != 0 {
		t.Errorf("err = %v, reads = %d; want ErrPlanUpgradeRequired and no read", err, reader.asked)
	}

	// The plan is checked before the period: a Starter never learns whether its period was valid.
	_, err = queries.NewGetConversion(gate, reader, clock).Handle(t.Context(), 7, 14)
	if !errors.Is(err, benefits.ErrPlanUpgradeRequired) {
		t.Errorf("invalid period on Starter: err = %v, want ErrPlanUpgradeRequired", err)
	}
	_, err = queries.NewGetConversion(proGate(), reader, clock).Handle(t.Context(), 7, 14)
	if !errors.Is(err, benefits.ErrMetricsPeriodInvalid) || reader.asked != 0 {
		t.Errorf("invalid period on Pro: err = %v, reads = %d; want ErrMetricsPeriodInvalid and no read", err, reader.asked)
	}
}
