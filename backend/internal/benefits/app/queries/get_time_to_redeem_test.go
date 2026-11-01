package queries_test

import (
	"context"
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/queries"
)

type fakeTimeToRedeem struct {
	got   benefits.TimeToRedeem
	asked int
}

func (f *fakeTimeToRedeem) TimeToRedeem(context.Context, int64, benefits.MetricsPeriod, benefits.MetricsPeriod) (benefits.TimeToRedeem, error) {
	f.asked++
	return f.got, nil
}

func TestTimeToRedeemSharesAddUpAndCompareWithThePreviousPeriod(t *testing.T) {
	t.Parallel()

	reader := &fakeTimeToRedeem{got: benefits.TimeToRedeem{
		Current:          benefits.RedeemTimes{SameDay: 6, Within24: 3, Within48: 1},
		Previous:         benefits.RedeemTimes{SameDay: 2, Within24: 4, Within48: 4},
		CouponsGenerated: 30, PreviousCouponsGenerated: 25,
	}}
	got, err := queries.NewGetTimeToRedeem(proGate(), reader, clock).Handle(t.Context(), 7, 30)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != queries.StatusOK || got.Redemptions != 10 || got.PreviousRedemptions != 10 {
		t.Fatalf("result = %+v, want ok with 10 redemptions against 10", got)
	}
	near(t, "same day", got.SameDay.Value, 0.6)
	near(t, "same day before", got.SameDay.Previous, 0.2)
	near(t, "same day change", got.SameDay.Change, 0.4)
	near(t, "24 hours", got.Within24h.Value, 0.3)
	near(t, "48 hours", got.Within48h.Value, 0.1)
	near(t, "48 hours change", got.Within48h.Change, -0.3)
	if sum := *got.SameDay.Value + *got.Within24h.Value + *got.Within48h.Value; sum < 0.9999 || sum > 1.0001 {
		t.Errorf("shares add up to %v, want 1", sum)
	}
}

func TestTimeToRedeemWithNoRedemptionsHasNoShareAndNeedsTheSample(t *testing.T) {
	t.Parallel()

	reader := &fakeTimeToRedeem{got: benefits.TimeToRedeem{CouponsGenerated: 25, PreviousCouponsGenerated: 25}}
	got, err := queries.NewGetTimeToRedeem(proGate(), reader, clock).Handle(t.Context(), 7, 30)
	if err != nil || got.Status != queries.StatusOK || got.SameDay.Value != nil || got.Redemptions != 0 {
		t.Errorf("no redemptions = %+v, %v; want ok with no share to divide", got, err)
	}

	reader.got = benefits.TimeToRedeem{Current: benefits.RedeemTimes{SameDay: 5}, CouponsGenerated: 19}
	got, _ = queries.NewGetTimeToRedeem(proGate(), reader, clock).Handle(t.Context(), 7, 30)
	if got.Status != queries.StatusInsufficientData || got.SameDay.Value != nil || got.CouponsGenerated != 19 {
		t.Errorf("19 coupons = %+v, want insufficient_data and no share", got)
	}

	reader.got = benefits.TimeToRedeem{
		Current: benefits.RedeemTimes{SameDay: 5}, Previous: benefits.RedeemTimes{SameDay: 5},
		CouponsGenerated: 20, PreviousCouponsGenerated: 19,
	}
	got, _ = queries.NewGetTimeToRedeem(proGate(), reader, clock).Handle(t.Context(), 7, 30)
	if got.SameDay.Previous != nil || got.SameDay.Change != nil {
		t.Errorf("a previous period of 19 coupons gave a baseline: %+v", got.SameDay)
	}
}

func TestTimeToRedeemIsRefusedBeforeAnythingIsReadForAStarter(t *testing.T) {
	t.Parallel()

	reader := &fakeTimeToRedeem{}
	denied := queries.NewProGate(fakeOwners{business: 42}, &fakePermission{allowed: false})
	if _, err := queries.NewGetTimeToRedeem(denied, reader, clock).Handle(t.Context(), 7, 30); !errors.Is(err, benefits.ErrPlanUpgradeRequired) || reader.asked != 0 {
		t.Errorf("Starter: err = %v, reads = %d; want ErrPlanUpgradeRequired and no read", err, reader.asked)
	}
	if _, err := queries.NewGetTimeToRedeem(proGate(), reader, clock).Handle(t.Context(), 7, 14); !errors.Is(err, benefits.ErrMetricsPeriodInvalid) || reader.asked != 0 {
		t.Errorf("period 14: err = %v, reads = %d; want ErrMetricsPeriodInvalid and no read", err, reader.asked)
	}
}
