package postgres_test

import (
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/adapters/postgres"
)

func monthsByLabel(d benefits.RecordsData) map[string]int64 {
	out := map[string]int64{}
	for _, m := range d.Months {
		out[m.Month] = m.Redemptions
	}
	return out
}

// TestRecordsFilesEachRedemptionInTheCalendarMonthOfArgentina pins the edges: 23:59:59 on the
// last day of a month still belongs to it even though UTC is already in the next one; the first
// instant of the next month starts it; February has 29 days in 2028; the first instant of the
// current month is the current month's, not the ranking's; and the window opens on the first
// instant of March 2027.
func TestRecordsFilesEachRedemptionInTheCalendarMonthOfArgentina(t *testing.T) {
	t.Parallel()

	f := newConversionFixture(t, "studio")
	monthStart := time.Date(2028, 3, 1, 0, 0, 0, 0, argentina)
	windowFrom := monthStart.AddDate(0, -12, 0)
	at := func(y int, m time.Month, d, h, mi, s int) time.Time {
		return time.Date(y, m, d, h, mi, s, 0, argentina)
	}

	f.redemption(t, at(2028, time.January, 31, 23, 59, 59))  // January, though UTC says February 1st
	f.redemption(t, at(2028, time.February, 1, 0, 0, 0))     // February
	f.redemption(t, at(2028, time.February, 29, 23, 59, 59)) // February, the 29th of a leap year
	f.redemption(t, at(2028, time.March, 1, 0, 0, 0))        // the current month
	f.redemption(t, at(2028, time.March, 10, 12, 0, 0))      // the current month
	f.redemption(t, at(2027, time.March, 1, 0, 0, 0))        // the first instant of the window
	f.redemption(t, at(2027, time.February, 28, 23, 59, 59)) // just before the window

	got, err := postgres.NewCoupons(f.pool).Records(t.Context(), f.business, windowFrom, monthStart)
	if err != nil {
		t.Fatal(err)
	}
	months := monthsByLabel(got)
	want := map[string]int64{"2028-01": 1, "2028-02": 2, "2027-03": 1}
	if len(months) != len(want) {
		t.Fatalf("months = %v, want %v", months, want)
	}
	for label, n := range want {
		if months[label] != n {
			t.Errorf("%s = %d, want %d (all: %v)", label, months[label], n, months)
		}
	}
	if got.CurrentMonthRedemptions != 2 {
		t.Errorf("current month = %d, want the 2 of March 2028", got.CurrentMonthRedemptions)
	}
	if got.FirstCouponAt == nil || !got.FirstCouponAt.Equal(at(2027, time.February, 28, 21, 59, 59)) {
		t.Errorf("first coupon at %v, want the earliest one created (two hours before its redemption)", got.FirstCouponAt)
	}
}

func TestRecordsTakesTheStrongestCellsAndBenefitsFromTheWindowOnly(t *testing.T) {
	t.Parallel()

	f := newConversionFixture(t, "studio")
	monthStart := time.Date(2028, 3, 1, 0, 0, 0, 0, argentina)
	windowFrom := monthStart.AddDate(0, -12, 0)

	// Saturday January 15th at 15:00, twice; Monday January 17th at 09:00, once.
	f.redemption(t, time.Date(2028, time.January, 15, 15, 0, 0, 0, argentina))
	f.redemption(t, time.Date(2028, time.January, 15, 16, 0, 0, 0, argentina))
	f.redemption(t, time.Date(2028, time.January, 17, 9, 0, 0, 0, argentina))
	// The current month and a month before the window must not count.
	f.redemption(t, time.Date(2028, time.March, 4, 15, 0, 0, 0, argentina))
	f.redemption(t, time.Date(2027, time.January, 15, 15, 0, 0, 0, argentina))

	got, err := postgres.NewCoupons(f.pool).Records(t.Context(), f.business, windowFrom, monthStart)
	if err != nil {
		t.Fatal(err)
	}
	cells := map[[2]int]int64{}
	for _, c := range got.Cells {
		cells[[2]int{c.Weekday, c.Slot}] = c.Redemptions
	}
	if len(cells) != 2 || cells[[2]int{6, 2}] != 2 || cells[[2]int{1, 1}] != 1 {
		t.Errorf("cells = %v, want Saturday afternoon twice and Monday morning once", cells)
	}
	if len(got.Benefits) != 1 || got.Benefits[0].Redemptions != 3 {
		t.Errorf("benefits = %+v, want the one benefit with the 3 redemptions of the window", got.Benefits)
	}
}

func TestRecordsOfABusinessWithoutCouponsAreEmptyAndNeverShowAnotherBusiness(t *testing.T) {
	t.Parallel()

	f := newConversionFixture(t, "studio")
	monthStart := time.Date(2028, 3, 1, 0, 0, 0, 0, argentina)
	windowFrom := monthStart.AddDate(0, -12, 0)

	other := newConversionFixtureIn(t, f, "other")
	other.redemption(t, time.Date(2028, time.January, 15, 15, 0, 0, 0, argentina))
	other.redemption(t, time.Date(2028, time.March, 4, 15, 0, 0, 0, argentina))

	got, err := postgres.NewCoupons(f.pool).Records(t.Context(), f.business, windowFrom, monthStart)
	if err != nil {
		t.Fatal(err)
	}
	if got.FirstCouponAt != nil || len(got.Months) != 0 || got.CurrentMonthRedemptions != 0 || len(got.Cells) != 0 || len(got.Benefits) != 0 {
		t.Errorf("records = %+v, want nothing for a business with no coupons", got)
	}
}
