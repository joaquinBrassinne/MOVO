package queries_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/queries"
)

type fakeRecords struct {
	got        benefits.RecordsData
	asked      int
	windowFrom time.Time
	monthStart time.Time
}

func (f *fakeRecords) Records(_ context.Context, _ int64, windowFrom, monthStart time.Time) (benefits.RecordsData, error) {
	f.asked++
	f.windowFrom, f.monthStart = windowFrom, monthStart
	return f.got, nil
}

var artZone = time.FixedZone("ART", -3*60*60)

func firstCoupon(y int, m time.Month, d int) *time.Time {
	t := time.Date(y, m, d, 0, 0, 0, 0, artZone)
	return &t
}

func months(pairs ...any) []benefits.MonthRedemptions {
	var out []benefits.MonthRedemptions
	for i := 0; i < len(pairs); i += 2 {
		out = append(out, benefits.MonthRedemptions{Month: pairs[i].(string), Redemptions: int64(pairs[i+1].(int))})
	}
	return out
}

// "now" of these tests is March 15th 2028 in Argentina, so the 12 months are March 2027 to February 2028.
func TestRecordsRankTheLastTwelveCompleteMonthsAndLeaveTheCurrentOneOut(t *testing.T) {
	t.Parallel()

	reader := &fakeRecords{got: benefits.RecordsData{
		FirstCouponAt: firstCoupon(2027, time.March, 5),
		// 2028-03 is the month in course: it must not enter the ranking even if it were sent.
		Months:                  months("2027-11", 25, "2027-12", 30, "2028-02", 22, "2028-03", 99),
		CurrentMonthRedemptions: 20,
		Cells:                   []benefits.HeatCell{{Weekday: 6, Slot: 2, Redemptions: 40}, {Weekday: 2, Slot: 1, Redemptions: 15}, {Weekday: 6, Slot: 1, Redemptions: 10}},
		Benefits:                []benefits.BenefitRedemptions{{BenefitID: 1, Redemptions: 20}, {BenefitID: 2, Redemptions: 45}},
	}}
	got, err := queries.NewGetRecords(proGate(), ownedBenefits(), reader, clock).Handle(t.Context(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != queries.StatusOK || got.BestMonth.Month != "2027-12" || got.BestMonth.Redemptions != 30 {
		t.Fatalf("result = %+v, want ok with 2027-12 and 30 redemptions as the best month", got)
	}
	if got.CurrentMonth.Month != "2028-03" || got.CurrentMonth.Redemptions != 20 || got.CurrentMonth.Progress != 0.6667 || got.CurrentMonth.RecordInProgress {
		t.Errorf("current month = %+v, want 2028-03 at 20 of 30 (0.6667), no record yet", got.CurrentMonth)
	}
	if got.StrongWeekday != 6 || got.StrongSlot != 2 || got.StrongSlotKey != "afternoon" {
		t.Errorf("strong = weekday %d slot %d %q, want Saturday in the afternoon", got.StrongWeekday, got.StrongSlot, got.StrongSlotKey)
	}
	if got.StarBenefit == nil || got.StarBenefit.BenefitID != 2 || got.StarBenefit.Name != "Clase de prueba" || got.StarBenefit.Redemptions != 45 {
		t.Errorf("star benefit = %+v, want the titled one with 45", got.StarBenefit)
	}
	if got.WindowFrom != "2027-03-01" || got.WindowTo != "2028-02-29" {
		t.Errorf("window = %s..%s, want 2027-03-01..2028-02-29", got.WindowFrom, got.WindowTo)
	}
	if reader.windowFrom.Format(time.DateOnly) != "2027-03-01" || reader.monthStart.Format(time.DateOnly) != "2028-03-01" {
		t.Errorf("asked from %v to %v, want March 1st 2027 to March 1st 2028", reader.windowFrom, reader.monthStart)
	}
}

func TestRecordsATieGoesToTheMostRecentMonth(t *testing.T) {
	t.Parallel()

	reader := &fakeRecords{got: benefits.RecordsData{
		FirstCouponAt: firstCoupon(2027, time.March, 5),
		Months:        months("2027-04", 30, "2027-12", 30, "2028-01", 30, "2028-02", 29),
	}}
	got, _ := queries.NewGetRecords(proGate(), ownedBenefits(), reader, clock).Handle(t.Context(), 7)
	if got.BestMonth.Month != "2028-01" {
		t.Errorf("best month = %s, want 2028-01, the most recent of the three tied at 30", got.BestMonth.Month)
	}
}

func TestRecordsSaysRecordInProgressOnceTheMonthBeatsTheBest(t *testing.T) {
	t.Parallel()

	data := benefits.RecordsData{FirstCouponAt: firstCoupon(2027, time.March, 5), Months: months("2028-02", 30)}
	for _, tc := range []struct {
		current  int64
		progress float64
		record   bool
	}{{15, 0.5, false}, {30, 1, false}, {31, 1.0333, true}, {45, 1.5, true}} {
		data.CurrentMonthRedemptions = tc.current
		got, _ := queries.NewGetRecords(proGate(), ownedBenefits(), &fakeRecords{got: data}, clock).Handle(t.Context(), 7)
		if got.CurrentMonth.Progress != tc.progress || got.CurrentMonth.RecordInProgress != tc.record {
			t.Errorf("%d of 30 = progress %v record %t, want %v and %t", tc.current, got.CurrentMonth.Progress, got.CurrentMonth.RecordInProgress, tc.progress, tc.record)
		}
	}
}

func TestRecordsNeedTwoCompleteMonthsOfHistoryAndTwentyRedemptionsInTheBestMonth(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		first   *time.Time
		months  []benefits.MonthRedemptions
		status  string
		history int
	}{
		{"never generated a coupon", nil, months("2028-02", 40), queries.StatusInsufficientData, 0},
		{"first coupon this month", firstCoupon(2028, time.March, 1), months("2028-02", 40), queries.StatusInsufficientData, 0},
		{"first coupon in the middle of February", firstCoupon(2028, time.February, 10), months("2028-02", 40), queries.StatusInsufficientData, 0},
		{"one complete month", firstCoupon(2028, time.January, 15), months("2028-02", 40), queries.StatusInsufficientData, 1},
		{"two complete months, first instant of January", firstCoupon(2028, time.January, 1), months("2028-01", 20), queries.StatusOK, 2},
		{"best month one short of twenty", firstCoupon(2027, time.March, 5), months("2027-12", 19, "2028-01", 12), queries.StatusInsufficientData, 11},
		{"nothing redeemed", firstCoupon(2027, time.March, 5), nil, queries.StatusInsufficientData, 11},
	} {
		got, err := queries.NewGetRecords(proGate(), ownedBenefits(), &fakeRecords{got: benefits.RecordsData{FirstCouponAt: tc.first, Months: tc.months}}, clock).Handle(t.Context(), 7)
		if err != nil || got.Status != tc.status || got.HistoryMonths != tc.history {
			t.Errorf("%s: status %q history %d (%v), want %q and %d", tc.name, got.Status, got.HistoryMonths, err, tc.status, tc.history)
		}
		if tc.status == queries.StatusInsufficientData && (got.CurrentMonth.Month != "" || got.StarBenefit != nil || got.StrongWeekday != 0) {
			t.Errorf("%s: shows records with insufficient data: %+v", tc.name, got)
		}
	}
}

func TestRecordsBreakTiesTowardTheEarlierWeekdayAndSlotAndTheOldestBenefit(t *testing.T) {
	t.Parallel()

	reader := &fakeRecords{got: benefits.RecordsData{
		FirstCouponAt: firstCoupon(2027, time.March, 5), Months: months("2028-02", 30),
		Cells:    []benefits.HeatCell{{Weekday: 5, Slot: 3, Redemptions: 10}, {Weekday: 3, Slot: 1, Redemptions: 10}, {Weekday: 3, Slot: 2, Redemptions: 5}},
		Benefits: []benefits.BenefitRedemptions{{BenefitID: 2, Redemptions: 12}, {BenefitID: 1, Redemptions: 12}},
	}}
	got, _ := queries.NewGetRecords(proGate(), ownedBenefits(), reader, clock).Handle(t.Context(), 7)
	// Weekday 3 has 15, weekday 5 has 10; slot 1 has 10, slot 3 has 10 and slot 2 has 5: the earlier slot wins.
	if got.StrongWeekday != 3 || got.StrongSlot != 1 {
		t.Errorf("strong = weekday %d slot %d, want 3 and 1", got.StrongWeekday, got.StrongSlot)
	}
	if got.StarBenefit == nil || got.StarBenefit.BenefitID != 1 || got.StarBenefit.Name != "-15%" {
		t.Errorf("star benefit = %+v, want the oldest of the two tied, named by its label", got.StarBenefit)
	}
}

func TestRecordsAreRefusedBeforeAnythingIsReadForAStarter(t *testing.T) {
	t.Parallel()

	reader := &fakeRecords{}
	denied := queries.NewProGate(fakeOwners{business: 42}, &fakePermission{allowed: false})
	if _, err := queries.NewGetRecords(denied, ownedBenefits(), reader, clock).Handle(t.Context(), 7); !errors.Is(err, benefits.ErrPlanUpgradeRequired) || reader.asked != 0 {
		t.Errorf("Starter: err = %v, reads = %d; want ErrPlanUpgradeRequired and no read", err, reader.asked)
	}
}
