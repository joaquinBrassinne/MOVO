package postgres_test

import (
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/adapters/postgres"
)

// couponsOf seeds n coupons created at createdAt, the first redeemed of them redeemed.
func (f *conversionFixture) couponsOf(t *testing.T, createdAt time.Time, n, redeemed int) {
	t.Helper()
	for i := range n {
		f.coupon(t, createdAt, i < redeemed, f.branch)
	}
}

func TestLowRedemptionReturnsTheActiveBusinessesUnderTheRateWithEnoughCoupons(t *testing.T) {
	t.Parallel()

	from := time.Date(2028, 2, 15, 0, 0, 0, 0, argentina)
	to := time.Date(2028, 3, 16, 0, 0, 0, 0, argentina)

	low := newConversionFixture(t, "low")
	low.couponsOf(t, from, 10, 1)                   // the first instant of the window counts
	low.couponsOf(t, to.Add(-time.Second), 10, 1)   // the last second counts
	low.couponsOf(t, from.Add(-time.Second), 30, 0) // just before the window
	low.couponsOf(t, to, 30, 0)                     // the first instant after it
	healthy := newConversionFixtureIn(t, low, "healthy")
	healthy.n = 100
	healthy.couponsOf(t, from.AddDate(0, 0, 3), 25, 10) // 0.4
	edge := newConversionFixtureIn(t, low, "edge")
	edge.n = 200
	edge.couponsOf(t, from.AddDate(0, 0, 3), 20, 4) // exactly 0.2 is not under it
	few := newConversionFixtureIn(t, low, "few")
	few.n = 300
	few.couponsOf(t, from.AddDate(0, 0, 3), 19, 0) // not enough coupons
	paused := newConversionFixtureIn(t, low, "paused")
	paused.n = 400
	paused.couponsOf(t, from.AddDate(0, 0, 3), 40, 0)
	if _, err := low.pool.Exec(t.Context(), `UPDATE businesses SET status = 'paused' WHERE id = $1`, paused.business); err != nil {
		t.Fatal(err)
	}

	got, err := postgres.NewCoupons(low.pool).LowRedemption(t.Context(), from, to, 20, 0.2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].BusinessID != low.business || got[0].Generated != 20 || got[0].Redeemed != 2 {
		t.Errorf("rates = %+v, want only business %d with 2 redeemed of 20", got, low.business)
	}
}
