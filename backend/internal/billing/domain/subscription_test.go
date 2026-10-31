package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/billing/domain"
)

func day(t *testing.T, s string) domain.Day {
	t.Helper()
	parsed, err := time.Parse(time.DateOnly, s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	d, err := domain.NewDay(parsed.Year(), parsed.Month(), parsed.Day())
	if err != nil {
		t.Fatalf("new day %q: %v", s, err)
	}
	return d
}

func sub(t *testing.T, paidUntil string) domain.Subscription {
	t.Helper()
	s, err := domain.NewSubscription(1, domain.PlanStarter, day(t, paidUntil))
	if err != nil {
		t.Fatalf("new subscription: %v", err)
	}
	return s
}

func TestStatusAtFollowsPaidUntilAndGrace(t *testing.T) {
	t.Parallel()

	s := sub(t, "2026-11-10")
	for _, tc := range []struct {
		today string
		want  domain.Status
	}{
		{"2026-10-01", domain.StatusActive},
		{"2026-11-10", domain.StatusActive}, // paid_until itself is still covered
		{"2026-11-11", domain.StatusGrace},
		{"2026-11-17", domain.StatusGrace}, // paid_until + 7 is the last day of grace
		{"2026-11-18", domain.StatusExpired},
		{"2027-01-01", domain.StatusExpired},
	} {
		if got := s.StatusAt(day(t, tc.today)); got != tc.want {
			t.Errorf("today %s: status = %q, want %q", tc.today, got, tc.want)
		}
	}
}

func TestVisibility(t *testing.T) {
	t.Parallel()

	today := day(t, "2026-11-12")
	active, grace, expired := sub(t, "2026-11-30"), sub(t, "2026-11-10"), sub(t, "2026-11-01")

	for _, tc := range []struct {
		name  string
		pilot bool
		sub   *domain.Subscription
		want  bool
	}{
		{"active subscription", false, &active, true},
		{"subscription in grace", false, &grace, true},
		{"expired subscription", false, &expired, false},
		{"no subscription", false, nil, false},
		{"pilot without subscription", true, nil, true},
		{"pilot with expired subscription", true, &expired, true},
	} {
		if got := domain.Visible(tc.pilot, tc.sub, today); got != tc.want {
			t.Errorf("%s: visible = %t, want %t", tc.name, got, tc.want)
		}
	}
}

func TestStatusOfPutsPilotsAndMissingSubscriptionsApart(t *testing.T) {
	t.Parallel()

	today := day(t, "2026-11-12")
	expired := sub(t, "2026-01-01")
	if got := domain.StatusOf(true, &expired, today); got != domain.StatusPilot {
		t.Errorf("pilot status = %q, want pilot", got)
	}
	if got := domain.StatusOf(false, nil, today); got != domain.StatusNone {
		t.Errorf("no subscription status = %q, want none", got)
	}
}

func TestRegisterPaymentExtendsOneMonth(t *testing.T) {
	t.Parallel()

	today := day(t, "2026-11-12")
	for _, tc := range []struct {
		name      string
		paidUntil string
		want      string
	}{
		{"current extends from paid_until", "2026-11-20", "2026-12-20"},
		{"paid early loses nothing", "2026-12-31", "2027-01-31"},
		{"lapsed extends from today", "2026-10-01", "2026-12-12"},
		{"in grace extends from today", "2026-11-10", "2026-12-12"},
		{"month end is clamped", "2027-01-31", "2027-02-28"},
		{"leap year clamp", "2028-01-31", "2028-02-29"},
	} {
		got := sub(t, tc.paidUntil).RegisterPayment(today)
		if got.PaidUntil.String() != tc.want {
			t.Errorf("%s: paid_until = %s, want %s", tc.name, got.PaidUntil, tc.want)
		}
	}
}

func TestExpiringSoonCoversTheNextSevenDaysOfAnActiveSubscription(t *testing.T) {
	t.Parallel()

	today := day(t, "2026-11-12")
	for _, tc := range []struct {
		paidUntil string
		want      bool
	}{
		{"2026-11-12", true}, // ends today
		{"2026-11-19", true}, // 7 days out
		{"2026-11-20", false},
		{"2026-11-11", false}, // already in grace: late, not "por vencer"
	} {
		if got := sub(t, tc.paidUntil).ExpiringSoon(today); got != tc.want {
			t.Errorf("paid_until %s: expiring soon = %t, want %t", tc.paidUntil, got, tc.want)
		}
	}
}

func TestChangePlanKeepsTheDatesAndRejectsUnknownPlans(t *testing.T) {
	t.Parallel()

	s := sub(t, "2026-11-30")
	changed, err := s.ChangePlan(domain.PlanPro)
	if err != nil {
		t.Fatalf("change plan: %v", err)
	}
	if changed.Plan != domain.PlanPro || changed.PaidUntil != s.PaidUntil {
		t.Errorf("changed = %+v, want pro with the same paid_until", changed)
	}
	if _, err := s.ChangePlan("gold"); !errors.Is(err, domain.ErrPlanInvalid) {
		t.Errorf("unknown plan error = %v, want ErrPlanInvalid", err)
	}
}

func TestNewSubscriptionRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	if _, err := domain.NewSubscription(1, "gold", day(t, "2026-11-30")); !errors.Is(err, domain.ErrPlanInvalid) {
		t.Errorf("unknown plan error = %v, want ErrPlanInvalid", err)
	}
	if _, err := domain.NewSubscription(1, domain.PlanPro, domain.Day{}); !errors.Is(err, domain.ErrPaidUntilInvalid) {
		t.Errorf("zero date error = %v, want ErrPaidUntilInvalid", err)
	}
	if _, err := domain.NewDay(2026, time.February, 30); !errors.Is(err, domain.ErrPaidUntilInvalid) {
		t.Errorf("Feb 30 error = %v, want ErrPaidUntilInvalid", err)
	}
}

func TestDayOfUsesArgentinaTime(t *testing.T) {
	t.Parallel()

	// 01:30 UTC on Nov 12 is still Nov 11 at 22:30 in Argentina.
	if got := domain.DayOf(time.Date(2026, 11, 12, 1, 30, 0, 0, time.UTC)).String(); got != "2026-11-11" {
		t.Errorf("DayOf = %s, want 2026-11-11", got)
	}
}
