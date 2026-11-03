package postgres_test

import (
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/adapters/postgres"
)

func TestRedemptionTotalsCountEachPeriodApartAtTheEdgesOfTheDay(t *testing.T) {
	t.Parallel()

	f := newConversionFixture(t, "studio")
	current := periodEnding(time.Date(2028, 3, 15, 0, 0, 0, 0, argentina), 7)
	previous := periodEnding(time.Date(2028, 3, 8, 0, 0, 0, 0, argentina), 7)
	at := func(d, h, m, s int) time.Time { return time.Date(2028, 3, d, h, m, s, 0, argentina) }

	// Redeemed at the first instant of the period, at its last second, one second before it and
	// at the instant it ends. Each is generated an hour before, but not before 00:00 of the day.
	f.redeemedAfter(t, at(9, 0, 0, 0).Add(-time.Hour), at(9, 0, 0, 0))
	f.redeemedAfter(t, at(15, 23, 0, 0), at(15, 23, 59, 59))
	f.redeemedAfter(t, at(8, 23, 0, 0), at(8, 23, 59, 59))
	f.redeemedAfter(t, at(15, 23, 30, 0), at(16, 0, 0, 0))
	f.coupon(t, at(12, 10, 0, 0), false, f.branch) // generated, not redeemed

	got, err := postgres.NewCoupons(f.pool).RedemptionTotals(t.Context(), f.business, current, previous)
	if err != nil {
		t.Fatal(err)
	}
	// Redemptions: the first two in the period; the 23:59:59 of March 8 in the previous one; the
	// instant March 16 00:00 in none. Generated: the coupons created inside each window.
	want := benefits.RedemptionTotals{Redemptions: 2, PreviousRedemptions: 1, CouponsGenerated: 3, PreviousCouponsGenerated: 2}
	if got != want {
		t.Errorf("totals = %+v, want %+v", got, want)
	}
}

func TestRedemptionTotalsLeaveOtherBusinessesOut(t *testing.T) {
	t.Parallel()

	f := newConversionFixture(t, "studio")
	current := periodEnding(time.Date(2028, 3, 15, 0, 0, 0, 0, argentina), 7)
	previous := periodEnding(time.Date(2028, 3, 8, 0, 0, 0, 0, argentina), 7)
	when := time.Date(2028, 3, 12, 10, 0, 0, 0, argentina)

	f.redeemedAfter(t, when, when.Add(time.Hour))
	other := newConversionFixtureIn(t, f, "other")
	other.redeemedAfter(t, when, when.Add(time.Hour))
	other.redeemedAfter(t, when, when.Add(time.Hour))

	got, err := postgres.NewCoupons(f.pool).RedemptionTotals(t.Context(), f.business, current, previous)
	if err != nil {
		t.Fatal(err)
	}
	if got.Redemptions != 1 || got.CouponsGenerated != 1 {
		t.Errorf("totals = %+v, want only this business's one redemption", got)
	}
}
