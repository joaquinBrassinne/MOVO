package commands_test

import (
	"context"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/commands"
)

type fakeLowRedemption struct {
	rates      []benefits.CouponRate
	from, to   time.Time
	minCoupons int
	below      float64
}

func (f *fakeLowRedemption) LowRedemption(_ context.Context, from, to time.Time, minCoupons int, below float64) ([]benefits.CouponRate, error) {
	f.from, f.to, f.minCoupons, f.below = from, to, minCoupons, below
	return f.rates, nil
}

type fakePro map[int64]bool

func (f fakePro) AdvancedMetrics(_ context.Context, businessID int64) (bool, error) {
	return f[businessID], nil
}

type raised struct {
	business int64
	kind     string
	payload  map[string]any
}

type fakeRaiser struct {
	already   map[int64]bool // businesses that got one this week
	notBefore time.Time
	raised    []raised
}

func (f *fakeRaiser) Raise(_ context.Context, businessID int64, kind string, payload map[string]any, notBefore time.Time) (bool, error) {
	f.notBefore = notBefore
	if f.already[businessID] {
		return false, nil
	}
	f.raised = append(f.raised, raised{businessID, kind, payload})
	return true, nil
}

// "now" is Wednesday March 15th 2028, 10:00 in Argentina.
var alertsNow = time.Date(2028, time.March, 15, 10, 0, 0, 0, time.FixedZone("ART", -3*60*60))

func newRaise(scanner *fakeLowRedemption, pro fakePro, raiser *fakeRaiser) *commands.RaiseUnredeemedAlerts {
	return commands.NewRaiseUnredeemedAlerts(scanner, pro, raiser, 20, 0.2, func() time.Time { return alertsNow })
}

func TestRaiseUnredeemedAlertsOnlyTellsTheBusinessesWithProMetrics(t *testing.T) {
	t.Parallel()

	scanner := &fakeLowRedemption{rates: []benefits.CouponRate{
		{BusinessID: 1, Generated: 40, Redeemed: 4},
		{BusinessID: 2, Generated: 50, Redeemed: 0},
		{BusinessID: 3, Generated: 25, Redeemed: 2},
	}}
	raiser := &fakeRaiser{}
	n, err := newRaise(scanner, fakePro{1: true, 3: true}, raiser).Handle(t.Context())
	if err != nil || n != 2 || len(raiser.raised) != 2 {
		t.Fatalf("Handle = %d, %v; raised %+v; want the two Pro businesses told (Starter 2 left out)", n, err, raiser.raised)
	}
	first := raiser.raised[0]
	if first.business != 1 || first.kind != benefits.AlertUnredeemedCoupons {
		t.Errorf("first = %+v, want business 1 and type %q", first, benefits.AlertUnredeemedCoupons)
	}
	want := map[string]any{"period_days": 30, "coupons_generated": int64(40), "coupons_redeemed": int64(4), "redeem_rate": 0.1, "threshold": 0.2}
	for key, value := range want {
		if first.payload[key] != value {
			t.Errorf("payload[%s] = %v, want %v (payload %v)", key, first.payload[key], value, first.payload)
		}
	}
	if len(first.payload) != len(want) {
		t.Errorf("payload = %v, want counts and rates only", first.payload)
	}
}

func TestRaiseUnredeemedAlertsLooksAtTheLastThirtyDaysWithTheConfiguredThresholds(t *testing.T) {
	t.Parallel()

	scanner := &fakeLowRedemption{}
	raiser := &fakeRaiser{}
	if _, err := newRaise(scanner, fakePro{}, raiser).Handle(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got := scanner.from.Format("2006-01-02 15:04 -0700"); got != "2028-02-15 00:00 -0300" {
		t.Errorf("window opens %s, want February 15th 00:00 (30 days with today)", got)
	}
	if got := scanner.to.Format("2006-01-02 15:04 -0700"); got != "2028-03-16 00:00 -0300" {
		t.Errorf("window closes %s, want the start of March 16th", got)
	}
	if scanner.minCoupons != 20 || scanner.below != 0.2 {
		t.Errorf("asked for %d coupons below %v, want 20 and 0.2", scanner.minCoupons, scanner.below)
	}
}

func TestRaiseUnredeemedAlertsAtMostOncePerWeek(t *testing.T) {
	t.Parallel()

	scanner := &fakeLowRedemption{rates: []benefits.CouponRate{{BusinessID: 1, Generated: 40, Redeemed: 4}, {BusinessID: 2, Generated: 40, Redeemed: 4}}}
	raiser := &fakeRaiser{already: map[int64]bool{1: true}}
	n, err := newRaise(scanner, fakePro{1: true, 2: true}, raiser).Handle(t.Context())
	if err != nil || n != 1 || len(raiser.raised) != 1 || raiser.raised[0].business != 2 {
		t.Fatalf("Handle = %d, %v; raised %+v; want only business 2, business 1 was told this week", n, err, raiser.raised)
	}
	if want := alertsNow.AddDate(0, 0, -7); !raiser.notBefore.Equal(want) {
		t.Errorf("notBefore = %v, want a week before now (%v)", raiser.notBefore, want)
	}
}
