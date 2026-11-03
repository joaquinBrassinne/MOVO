package postgres_test

import (
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/adapters/postgres"
)

// redemption inserts a coupon redeemed at the instant given. The fixture redeems a coupon two
// hours after creating it, so it is created two hours earlier.
func (f *conversionFixture) redemption(t *testing.T, redeemedAt time.Time) {
	t.Helper()
	f.coupon(t, redeemedAt.Add(-2*time.Hour), true, f.branch)
}

// newConversionFixtureIn adds another business to the database of base, with its own coupons.
func newConversionFixtureIn(t *testing.T, base *conversionFixture, name string) *conversionFixture {
	t.Helper()
	ctx := t.Context()
	s := newShop(ctx, t, base.pool, name, "active", 0)
	f := &conversionFixture{pool: base.pool, business: s.business, customer: base.customer, n: 500}
	if err := f.pool.QueryRow(ctx, `SELECT owner_user_id FROM businesses WHERE id = $1`, s.business).Scan(&f.owner); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(ctx, `SELECT id FROM branches WHERE business_id = $1`, s.business).Scan(&f.branch); err != nil {
		t.Fatal(err)
	}
	return f
}

func cellsOf(h benefits.Heatmap) map[[2]int]int64 {
	out := map[[2]int]int64{}
	for _, c := range h.Cells {
		out[[2]int{c.Weekday, c.Slot}] = c.Redemptions
	}
	return out
}

// TestHeatmapPutsEachRedemptionInTheWeekdayAndSlotOfArgentinasClock pins the edges: 23:59:59 of
// a Saturday is still Saturday's evening and 00:00:00 is Sunday's night; the slots change at 6,
// 12 and 18 o'clock sharp.
func TestHeatmapPutsEachRedemptionInTheWeekdayAndSlotOfArgentinasClock(t *testing.T) {
	t.Parallel()

	f := newConversionFixture(t, "studio")
	current := periodEnding(time.Date(2028, 3, 15, 0, 0, 0, 0, argentina), 7) // Thursday March 9 to Wednesday 15
	previous := periodEnding(time.Date(2028, 3, 8, 0, 0, 0, 0, argentina), 7) // March 2 to 8
	at := func(d, h, m, s int) time.Time { return time.Date(2028, 3, d, h, m, s, 0, argentina) }

	// Saturday March 11 into Sunday March 12.
	f.redemption(t, at(11, 23, 59, 59))
	f.redemption(t, at(12, 0, 0, 0))
	// Monday March 13 around the edges of the slots.
	for _, h := range []struct{ h, m, s int }{{5, 59, 59}, {6, 0, 0}, {11, 59, 59}, {12, 0, 0}, {17, 59, 59}, {18, 0, 0}} {
		f.redemption(t, at(13, h.h, h.m, h.s))
	}
	// The edges of the period: the last second before it belongs to the previous one, the
	// first instant after it to the next.
	f.redemption(t, at(8, 23, 59, 59))
	f.redemption(t, at(16, 0, 0, 0))
	f.redemption(t, at(9, 0, 0, 0)) // Thursday, first instant of the period

	got, err := postgres.NewCoupons(f.pool).Heatmap(t.Context(), f.business, current, previous)
	if err != nil {
		t.Fatal(err)
	}
	want := map[[2]int]int64{
		{6, 3}: 1, // Saturday 23:59:59
		{7, 0}: 1, // Sunday 00:00:00
		{1, 0}: 1, // Monday 05:59:59
		{1, 1}: 2, // Monday 06:00:00 and 11:59:59
		{1, 2}: 2, // Monday 12:00:00 and 17:59:59
		{1, 3}: 1, // Monday 18:00:00
		{4, 0}: 1, // Thursday 00:00:00, first instant of the period
	}
	cells := cellsOf(got)
	if len(cells) != len(want) {
		t.Fatalf("cells = %v, want %v", cells, want)
	}
	for k, n := range want {
		if cells[k] != n {
			t.Errorf("weekday %d slot %d = %d, want %d", k[0], k[1], cells[k], n)
		}
	}
	// The nine in the cells; the one at 23:59:59 of March 8 is in the previous period.
	if got.Redemptions != 9 || got.PreviousRedemptions != 1 {
		t.Errorf("totals = %d now, %d before; want 9 and 1", got.Redemptions, got.PreviousRedemptions)
	}
}

func TestHeatmapAcrossTheEndOfFebruaryInALeapYear(t *testing.T) {
	t.Parallel()

	f := newConversionFixture(t, "studio")
	current := periodEnding(time.Date(2028, 3, 2, 0, 0, 0, 0, argentina), 7) // February 25 to March 2
	previous := periodEnding(time.Date(2028, 2, 24, 0, 0, 0, 0, argentina), 7)

	f.redemption(t, time.Date(2028, 2, 29, 23, 59, 59, 0, argentina)) // Tuesday, evening
	f.redemption(t, time.Date(2028, 3, 1, 0, 0, 0, 0, argentina))     // Wednesday, night
	f.redemption(t, time.Date(2028, 2, 24, 23, 59, 59, 0, argentina)) // before the period

	got, err := postgres.NewCoupons(f.pool).Heatmap(t.Context(), f.business, current, previous)
	if err != nil {
		t.Fatal(err)
	}
	cells := cellsOf(got)
	if len(cells) != 2 || cells[[2]int{2, 3}] != 1 || cells[[2]int{3, 0}] != 1 {
		t.Errorf("cells = %v, want Tuesday evening and Wednesday night", cells)
	}
	if got.Redemptions != 2 || got.PreviousRedemptions != 1 {
		t.Errorf("totals = %d now, %d before; want 2 and 1", got.Redemptions, got.PreviousRedemptions)
	}
}

func TestHeatmapCountsOnlyThisBusinessAndOnlyRedeemedCoupons(t *testing.T) {
	t.Parallel()

	f := newConversionFixture(t, "studio")
	current := periodEnding(time.Date(2028, 3, 15, 0, 0, 0, 0, argentina), 7)
	previous := periodEnding(time.Date(2028, 3, 8, 0, 0, 0, 0, argentina), 7)
	inside := time.Date(2028, 3, 13, 10, 0, 0, 0, argentina)

	f.redemption(t, inside)
	f.coupon(t, inside, false, f.branch) // generated but never redeemed: it counts in the sample only

	other := newConversionFixtureIn(t, f, "other")
	other.redemption(t, inside)
	other.redemption(t, inside)

	got, err := postgres.NewCoupons(f.pool).Heatmap(t.Context(), f.business, current, previous)
	if err != nil {
		t.Fatal(err)
	}
	if got.Redemptions != 1 || got.CouponsGenerated != 2 || cellsOf(got)[[2]int{1, 1}] != 1 {
		t.Errorf("heatmap = %+v, want 1 redemption and 2 coupons generated for this business only", got)
	}
}
