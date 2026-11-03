package postgres_test

import (
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/adapters/postgres"
)

// redeemedAfter inserts a coupon generated at createdAt and redeemed at redeemedAt.
func (f *conversionFixture) redeemedAfter(t *testing.T, createdAt, redeemedAt time.Time) {
	t.Helper()
	f.n++
	code := raceCode(f.n)
	hash := make([]byte, 32)
	copy(hash, code)
	var benefit int64
	if err := f.pool.QueryRow(t.Context(), `SELECT id FROM benefits WHERE business_id = $1`, f.business).Scan(&benefit); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(t.Context(), `
INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, status, expires_at, created_at, redeemed_at, redeemed_by, redeemed_branch_id)
VALUES ($1, $2, $3, $4, $5, 'redeemed', $6, $7, $8, $9, $10)`,
		code, benefit, f.business, f.customer, hash, createdAt.Add(48*time.Hour), createdAt, redeemedAt, f.owner, f.branch); err != nil {
		t.Fatalf("insert redeemed coupon: %v", err)
	}
}

// TestTimeToRedeemSortsByCalendarDayThenByElapsedTime pins the edges: two minutes across
// midnight is a later day, not the same one; exactly 24 hours is still within 24 hours; one
// second more is within 48; and a day change is judged on Argentina's calendar, not UTC's.
func TestTimeToRedeemSortsByCalendarDayThenByElapsedTime(t *testing.T) {
	t.Parallel()

	f := newConversionFixture(t, "studio")
	current := periodEnding(time.Date(2028, 3, 15, 0, 0, 0, 0, argentina), 7)
	previous := periodEnding(time.Date(2028, 3, 8, 0, 0, 0, 0, argentina), 7)
	at := func(d, h, m, s int) time.Time { return time.Date(2028, 3, d, h, m, s, 0, argentina) }

	f.redeemedAfter(t, at(12, 0, 0, 0), at(12, 23, 59, 59)) // same day, almost 24 hours later
	f.redeemedAfter(t, at(12, 9, 0, 0), at(12, 9, 0, 1))    // same day
	f.redeemedAfter(t, at(12, 23, 59, 0), at(13, 0, 1, 0))  // two minutes later but the next day
	f.redeemedAfter(t, at(12, 10, 0, 0), at(13, 10, 0, 0))  // exactly 24 hours: within 24
	f.redeemedAfter(t, at(12, 10, 0, 0), at(13, 10, 0, 1))  // 24 hours and a second: within 48
	f.redeemedAfter(t, at(12, 10, 0, 0), at(14, 9, 0, 0))   // 47 hours: within 48
	// A coupon generated at 22:00 in Argentina is already the next day in UTC; the redemption
	// at 23:00 the same Argentine day is the same day, though UTC says otherwise.
	f.redeemedAfter(t, at(10, 22, 0, 0), at(10, 23, 0, 0))
	// Redeemed before the period or after it: not counted here.
	f.redeemedAfter(t, at(8, 10, 0, 0), at(8, 23, 59, 59))
	f.redeemedAfter(t, at(16, 0, 0, 0), at(16, 0, 1, 0))

	got, err := postgres.NewCoupons(f.pool).TimeToRedeem(t.Context(), f.business, current, previous)
	if err != nil {
		t.Fatal(err)
	}
	if want := (benefits.RedeemTimes{SameDay: 3, Within24: 2, Within48: 2}); got.Current != want {
		t.Errorf("current = %+v, want %+v", got.Current, want)
	}
	if want := (benefits.RedeemTimes{SameDay: 1}); got.Previous != want {
		t.Errorf("previous = %+v, want %+v", got.Previous, want)
	}
	// Generated in the period: the seven created from March 10 to 12; the one of March 8 is before it.
	if got.CouponsGenerated != 7 {
		t.Errorf("coupons generated = %d, want 7", got.CouponsGenerated)
	}
}

func TestTimeToRedeemIgnoresOtherBusinessesAndUnredeemedCoupons(t *testing.T) {
	t.Parallel()

	f := newConversionFixture(t, "studio")
	current := periodEnding(time.Date(2028, 3, 15, 0, 0, 0, 0, argentina), 7)
	previous := periodEnding(time.Date(2028, 3, 8, 0, 0, 0, 0, argentina), 7)
	created := time.Date(2028, 3, 12, 10, 0, 0, 0, argentina)

	f.redeemedAfter(t, created, created.Add(time.Hour))
	f.coupon(t, created, false, f.branch)
	other := newConversionFixtureIn(t, f, "other")
	other.redeemedAfter(t, created, created.Add(time.Hour))
	other.redeemedAfter(t, created, created.Add(time.Hour))

	got, err := postgres.NewCoupons(f.pool).TimeToRedeem(t.Context(), f.business, current, previous)
	if err != nil {
		t.Fatal(err)
	}
	if got.Current != (benefits.RedeemTimes{SameDay: 1}) || got.CouponsGenerated != 2 {
		t.Errorf("got %+v with %d coupons generated, want one same-day redemption of 2 coupons", got.Current, got.CouponsGenerated)
	}
}
