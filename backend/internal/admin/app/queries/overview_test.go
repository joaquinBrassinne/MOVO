package queries_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/admin"
	"github.com/joaquinBrassinne/MOVO/internal/admin/app/queries"
)

type reader struct {
	gotMonthStart time.Time
	overview      admin.Overview
	err           error
}

func (r *reader) Overview(_ context.Context, monthStart time.Time) (admin.Overview, error) {
	r.gotMonthStart = monthStart
	return r.overview, r.err
}

func TestMonthStartRollsOverOnArgentinasClock(t *testing.T) {
	t.Parallel()

	// 01:00 UTC on December 1st is still 22:00 of November 30th in Argentina.
	got := queries.MonthStart(time.Date(2026, 12, 1, 1, 0, 0, 0, time.UTC))
	if want := time.Date(2026, 11, 1, 3, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("month start = %v, want %v", got.UTC(), want)
	}

	// Three hours later it is December there too.
	got = queries.MonthStart(time.Date(2026, 12, 1, 3, 0, 0, 0, time.UTC))
	if want := time.Date(2026, 12, 1, 3, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("month start = %v, want %v", got.UTC(), want)
	}
}

func TestGetOverviewAsksForTheCurrentMonthAndReportsIt(t *testing.T) {
	t.Parallel()

	r := &reader{overview: admin.Overview{ActiveBusinesses: 7, Coupons: admin.CouponCounts{Generated: 10, Redeemed: 4}}}
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)

	got, err := queries.NewGetOverview(r, func() time.Time { return now }).Handle(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	want := queries.MonthStart(now)
	if !r.gotMonthStart.Equal(want) || !got.MonthStart.Equal(want) {
		t.Errorf("month start asked %v, reported %v, want %v", r.gotMonthStart, got.MonthStart, want)
	}
	if got.ActiveBusinesses != 7 || got.Coupons.Generated != 10 || got.Coupons.Redeemed != 4 {
		t.Errorf("overview = %+v, want the reader's figures", got)
	}
}

func TestGetOverviewWrapsTheReaderError(t *testing.T) {
	t.Parallel()

	boom := errors.New("down")
	_, err := queries.NewGetOverview(&reader{err: boom}, nil).Handle(t.Context())
	if !errors.Is(err, boom) {
		t.Errorf("error = %v, want it to wrap the reader's", err)
	}
}
