package queries_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/queries"
)

type fakeMetrics struct {
	got    benefits.EssentialMetrics
	period benefits.MetricsPeriod
	asked  []int64
}

func (f *fakeMetrics) Essential(_ context.Context, businessID int64, p benefits.MetricsPeriod) (benefits.EssentialMetrics, error) {
	f.asked = append(f.asked, businessID)
	f.period = p
	return f.got, nil
}

func TestMetricsReadTheBusinessOfTheSessionOverTheDefaultPeriod(t *testing.T) {
	t.Parallel()

	reader := &fakeMetrics{got: benefits.EssentialMetrics{ProfileViews: 12, CouponsGenerated: 5, CouponsRedeemed: 2}}
	got, err := queries.NewGetMetrics(fakeOwners{business: 42}, reader, clock).Handle(t.Context(), 7, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(reader.asked) != 1 || reader.asked[0] != 42 {
		t.Errorf("asked businesses %v, want only the owner's 42", reader.asked)
	}
	if got.Metrics != reader.got || got.Period.Days != queries.DefaultMetricsDays {
		t.Errorf("result = %+v, want the reader's figures over %d days", got, queries.DefaultMetricsDays)
	}
	// 30 days ending on March 15th of a leap year start on February 15th.
	if got.Period.FromDay != "2028-02-15" || got.Period.ToDay != "2028-03-15" {
		t.Errorf("window = %s..%s, want 2028-02-15..2028-03-15", got.Period.FromDay, got.Period.ToDay)
	}
}

func TestMetricsPeriodEndsWithTodayOnArgentinasCalendar(t *testing.T) {
	t.Parallel()

	// 02:00 UTC on March 16th is still 23:00 of March 15th in Argentina.
	late := func() time.Time { return time.Date(2028, 3, 16, 2, 0, 0, 0, time.UTC) }
	reader := &fakeMetrics{}
	if _, err := queries.NewGetMetrics(fakeOwners{business: 1}, reader, late).Handle(t.Context(), 7, 7); err != nil {
		t.Fatal(err)
	}
	p := reader.period
	if p.FromDay != "2028-03-09" || p.ToDay != "2028-03-15" {
		t.Errorf("window = %s..%s, want 2028-03-09..2028-03-15", p.FromDay, p.ToDay)
	}
	// The coupon tables use instants: the window closes at the next local midnight, 03:00 UTC.
	if want := time.Date(2028, 3, 16, 3, 0, 0, 0, time.UTC); !p.To.Equal(want) {
		t.Errorf("window closes at %v, want %v", p.To.UTC(), want)
	}
}

func TestMetricsOfferOnlySevenThirtyAndNinetyDays(t *testing.T) {
	t.Parallel()

	for _, days := range []int{7, 30, 90} {
		if _, err := queries.NewGetMetrics(fakeOwners{business: 1}, &fakeMetrics{}, clock).Handle(t.Context(), 1, days); err != nil {
			t.Errorf("days %d: %v", days, err)
		}
	}
	for _, days := range []int{-7, 1, 14, 31, 365} {
		reader := &fakeMetrics{}
		_, err := queries.NewGetMetrics(fakeOwners{business: 1}, reader, clock).Handle(t.Context(), 1, days)
		if !errors.Is(err, benefits.ErrMetricsPeriodInvalid) || len(reader.asked) != 0 {
			t.Errorf("days %d: err = %v, reads = %d; want ErrMetricsPeriodInvalid and no read", days, err, len(reader.asked))
		}
	}
}

func TestMetricsNeedABusiness(t *testing.T) {
	t.Parallel()

	reader := &fakeMetrics{}
	_, err := queries.NewGetMetrics(fakeOwners{err: benefits.ErrBusinessNotFound}, reader, clock).Handle(t.Context(), 1, 30)
	if !errors.Is(err, benefits.ErrBusinessNotFound) || len(reader.asked) != 0 {
		t.Errorf("err = %v, reads = %d; want ErrBusinessNotFound and no read", err, len(reader.asked))
	}
}
