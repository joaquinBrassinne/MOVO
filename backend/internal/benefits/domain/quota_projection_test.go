package domain_test

import (
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

var art = time.FixedZone("ART", -3*60*60)

func quota(limit *int, generated int) domain.Quota {
	return domain.Quota{Limit: limit, Generated: generated}
}

func limitOf(n int) *int { return &n }

func TestProjectQuotaEstimatesTheDayItRunsOut(t *testing.T) {
	t.Parallel()

	// November 2028 has 30 days. On the 10th, 40 coupons is 4 a day: a limit of 100 runs out
	// 15 days later, on the 25th.
	now := time.Date(2028, 11, 10, 15, 0, 0, 0, art)
	start, end := domain.QuotaMonth(now)
	got := domain.ProjectQuota(quota(limitOf(100), 40), start, end, now)
	if got.Status != domain.ProjectionWillRunOut || got.ExhaustsOn != "2028-11-25" || got.DailyRate != 4 || got.ProjectedTotal != 120 {
		t.Errorf("projection = %+v, want will_run_out on 2028-11-25 at 4 a day, 120 for the month", got)
	}
	if got.DaysElapsed != 10 || got.DaysInMonth != 30 || got.Limit != 100 || got.Generated != 40 {
		t.Errorf("projection = %+v, want 10 of 30 days, 40 of 100", got)
	}
}

func TestProjectQuotaRoundsTheDayUpAndStaysOnTrackWhenItLasts(t *testing.T) {
	t.Parallel()

	now := time.Date(2028, 11, 10, 15, 0, 0, 0, art)
	start, end := domain.QuotaMonth(now)

	// 30 in 10 days is 3 a day: 100 needs 70 more, 23.33 days, so the 24th day after the 10th.
	// That is past the 30th: on track.
	if got := domain.ProjectQuota(quota(limitOf(100), 30), start, end, now); got.Status != domain.ProjectionOnTrack || got.ExhaustsOn != "" || got.ProjectedTotal != 90 {
		t.Errorf("30 of 100 = %+v, want on_track ending the month at 90", got)
	}
	// 3 a day with a limit of 60: 30 more is exactly 10 days, the 20th.
	if got := domain.ProjectQuota(quota(limitOf(60), 30), start, end, now); got.Status != domain.ProjectionWillRunOut || got.ExhaustsOn != "2028-11-20" {
		t.Errorf("30 of 60 = %+v, want will_run_out on 2028-11-20", got)
	}
	// 31 in 10 days with a limit of 60: 29 more at 3.1 a day is 9.35 days, rounded up to 10: the 20th.
	if got := domain.ProjectQuota(quota(limitOf(60), 31), start, end, now); got.ExhaustsOn != "2028-11-20" {
		t.Errorf("31 of 60 = %+v, want the day rounded up to 2028-11-20", got)
	}
	// Running out exactly on the last day of the month still runs out.
	last := time.Date(2028, 11, 10, 15, 0, 0, 0, art)
	if got := domain.ProjectQuota(quota(limitOf(90), 30), start, end, last); got.Status != domain.ProjectionWillRunOut || got.ExhaustsOn != "2028-11-30" {
		t.Errorf("30 of 90 = %+v, want will_run_out on the 30th", got)
	}
}

func TestProjectQuotaOnlyAnswersWithAQuotaAndEnoughOfTheMonth(t *testing.T) {
	t.Parallel()

	now := time.Date(2028, 11, 10, 15, 0, 0, 0, art)
	start, end := domain.QuotaMonth(now)

	if got := domain.ProjectQuota(quota(nil, 500), start, end, now); got.Status != domain.ProjectionNoQuota {
		t.Errorf("no limit = %+v, want no_quota", got)
	}
	if got := domain.ProjectQuota(quota(limitOf(100), 100), start, end, now); got.Status != domain.ProjectionReached || got.ExhaustsOn != "" {
		t.Errorf("limit reached = %+v, want exhausted", got)
	}
	if got := domain.ProjectQuota(quota(limitOf(100), 0), start, end, now); got.Status != domain.ProjectionTooEarly {
		t.Errorf("nothing generated = %+v, want insufficient_data", got)
	}

	// Day 2 is too early, day 3 is the first with an estimate; a reached quota is exhausted whatever the day.
	second := time.Date(2028, 11, 2, 23, 59, 59, 0, art)
	if got := domain.ProjectQuota(quota(limitOf(100), 50), start, end, second); got.Status != domain.ProjectionTooEarly {
		t.Errorf("day 2 = %+v, want insufficient_data", got)
	}
	third := time.Date(2028, 11, 3, 0, 0, 0, 0, art)
	if got := domain.ProjectQuota(quota(limitOf(100), 50), start, end, third); got.Status != domain.ProjectionWillRunOut {
		t.Errorf("day 3 = %+v, want an estimate", got)
	}
	if got := domain.ProjectQuota(quota(limitOf(10), 10), start, end, second); got.Status != domain.ProjectionReached {
		t.Errorf("day 2 with the quota used up = %+v, want exhausted", got)
	}
}

func TestProjectQuotaReadsTheDayOnArgentinasClockAtTheEdgeOfTheMonth(t *testing.T) {
	t.Parallel()

	// 02:00 UTC on December 1st is still November 30th at 23:00 in Argentina: the 30th day.
	now := time.Date(2028, 12, 1, 2, 0, 0, 0, time.UTC)
	start, end := domain.QuotaMonth(now)
	if start.Month() != time.November {
		t.Fatalf("month starts %v, want November", start)
	}
	got := domain.ProjectQuota(quota(limitOf(100), 60), start, end, now)
	if got.DaysElapsed != 30 || got.DaysInMonth != 30 || got.Status != domain.ProjectionOnTrack || got.ProjectedTotal != 60 {
		t.Errorf("projection = %+v, want day 30 of 30 on track at 60", got)
	}

	// February in a leap year has 29 days.
	feb := time.Date(2028, 2, 15, 12, 0, 0, 0, art)
	fs, fe := domain.QuotaMonth(feb)
	if got := domain.ProjectQuota(quota(limitOf(1000), 15), fs, fe, feb); got.DaysInMonth != 29 || got.ProjectedTotal != 29 {
		t.Errorf("February = %+v, want 29 days and 29 projected", got)
	}
}
