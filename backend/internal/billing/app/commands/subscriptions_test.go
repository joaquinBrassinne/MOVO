package commands_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/billing"
	"github.com/joaquinBrassinne/MOVO/internal/billing/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/billing/domain"
)

// repo is an in-memory SubscriptionRepository that records the audit entries.
type repo struct {
	subs    map[int64]domain.Subscription
	audit   []billing.AuditEntry
	confirm error
}

func (r *repo) Find(_ context.Context, id int64) (domain.Subscription, error) {
	s, ok := r.subs[id]
	if !ok {
		return domain.Subscription{}, billing.ErrSubscriptionNotFound
	}
	return s, nil
}

func (r *repo) Confirm(_ context.Context, s domain.Subscription, _ int64, e billing.AuditEntry) error {
	if r.confirm != nil {
		return r.confirm
	}
	r.subs[s.BusinessID] = s
	r.audit = append(r.audit, e)
	return nil
}

func (r *repo) Update(_ context.Context, s, _ domain.Subscription, e billing.AuditEntry) error {
	r.subs[s.BusinessID] = s
	r.audit = append(r.audit, e)
	return nil
}

var now = time.Date(2026, 11, 12, 15, 0, 0, 0, time.UTC)

func clock() time.Time { return now }

func day(t *testing.T, s string) domain.Day {
	t.Helper()
	parsed, err := time.Parse(time.DateOnly, s)
	if err != nil {
		t.Fatal(err)
	}
	return domain.DayFromDate(parsed)
}

func TestConfirmPaymentStoresWhatTheAdminChoseAndAudits(t *testing.T) {
	t.Parallel()

	r := &repo{subs: map[int64]domain.Subscription{}}
	got, err := commands.NewConfirmPayment(r, clock).Handle(t.Context(), commands.ConfirmPaymentInput{
		ActorID: 7, BusinessID: 3, Plan: "pro", PaidUntil: day(t, "2026-12-12"), MPSubscriptionRef: " MP-9 ", Notes: "transfer",
	})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if got.Plan != domain.PlanPro || got.PaidUntil.String() != "2026-12-12" || got.MPSubscriptionRef != "MP-9" {
		t.Errorf("subscription = %+v", got)
	}
	if len(r.audit) != 1 || r.audit[0].Action != "billing.confirm_payment" || r.audit[0].ActorID != 7 || r.audit[0].BusinessID != 3 {
		t.Errorf("audit = %+v", r.audit)
	}
}

func TestConfirmPaymentRejectsBadInputBeforeTouchingTheRepository(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		in   commands.ConfirmPaymentInput
		want error
	}{
		{"unknown plan", commands.ConfirmPaymentInput{Plan: "gold", PaidUntil: day(t, "2026-12-12")}, domain.ErrPlanInvalid},
		{"no date", commands.ConfirmPaymentInput{Plan: "pro"}, domain.ErrPaidUntilInvalid},
		{"yesterday", commands.ConfirmPaymentInput{Plan: "pro", PaidUntil: day(t, "2026-11-11")}, commands.ErrPaidUntilInThePast},
		{"long reference", commands.ConfirmPaymentInput{Plan: "pro", PaidUntil: day(t, "2026-12-12"), MPSubscriptionRef: strings.Repeat("a", 101)}, domain.ErrRefInvalid},
		{"long notes", commands.ConfirmPaymentInput{Plan: "pro", PaidUntil: day(t, "2026-12-12"), Notes: strings.Repeat("a", 501)}, domain.ErrNotesInvalid},
	} {
		r := &repo{subs: map[int64]domain.Subscription{}, confirm: errors.New("must not be called")}
		if _, err := commands.NewConfirmPayment(r, clock).Handle(t.Context(), tc.in); !errors.Is(err, tc.want) {
			t.Errorf("%s: error = %v, want %v", tc.name, err, tc.want)
		}
	}

	// Today is a valid last day: the business is paid for the day it is confirmed.
	r := &repo{subs: map[int64]domain.Subscription{}}
	if _, err := commands.NewConfirmPayment(r, clock).Handle(t.Context(), commands.ConfirmPaymentInput{Plan: "pro", PaidUntil: day(t, "2026-11-12")}); err != nil {
		t.Errorf("paid until today: %v", err)
	}
}

func TestConfirmPaymentPassesTheRepositoryErrorsThrough(t *testing.T) {
	t.Parallel()

	r := &repo{subs: map[int64]domain.Subscription{}, confirm: billing.ErrSubscriptionExists}
	_, err := commands.NewConfirmPayment(r, clock).Handle(t.Context(), commands.ConfirmPaymentInput{Plan: "pro", PaidUntil: day(t, "2026-12-12")})
	if !errors.Is(err, billing.ErrSubscriptionExists) {
		t.Errorf("error = %v, want ErrSubscriptionExists", err)
	}
}

func TestRegisterPaymentExtendsAndAudits(t *testing.T) {
	t.Parallel()

	start, _ := domain.NewSubscription(3, domain.PlanStarter, day(t, "2026-11-20"))
	r := &repo{subs: map[int64]domain.Subscription{3: start}}

	got, err := commands.NewRegisterPayment(r, clock).Handle(t.Context(), 7, 3)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if got.PaidUntil.String() != "2026-12-20" {
		t.Errorf("paid_until = %s, want 2026-12-20", got.PaidUntil)
	}
	meta := r.audit[0].Metadata
	if r.audit[0].Action != "billing.register_payment" || meta["paid_until_from"] != "2026-11-20" || meta["paid_until_to"] != "2026-12-20" {
		t.Errorf("audit = %+v", r.audit[0])
	}

	if _, err := commands.NewRegisterPayment(&repo{subs: map[int64]domain.Subscription{}}, clock).Handle(t.Context(), 7, 99); !errors.Is(err, billing.ErrSubscriptionNotFound) {
		t.Errorf("no subscription error = %v, want ErrSubscriptionNotFound", err)
	}
}

func TestChangePlanKeepsTheDateAuditsAndRefusesNoOps(t *testing.T) {
	t.Parallel()

	start, _ := domain.NewSubscription(3, domain.PlanPro, day(t, "2026-11-20"))
	r := &repo{subs: map[int64]domain.Subscription{3: start}}
	change := commands.NewChangePlan(r)

	got, err := change.Handle(t.Context(), 7, 3, "starter")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if got.Plan != domain.PlanStarter || got.PaidUntil != start.PaidUntil {
		t.Errorf("subscription = %+v", got)
	}
	if r.audit[0].Action != "billing.change_plan" || r.audit[0].Metadata["plan_from"] != "pro" || r.audit[0].Metadata["plan_to"] != "starter" {
		t.Errorf("audit = %+v", r.audit[0])
	}
	if _, err := change.Handle(t.Context(), 7, 3, "starter"); !errors.Is(err, commands.ErrNoChange) {
		t.Errorf("same plan error = %v, want ErrNoChange", err)
	}
	if _, err := change.Handle(t.Context(), 7, 3, "gold"); !errors.Is(err, domain.ErrPlanInvalid) {
		t.Errorf("unknown plan error = %v, want ErrPlanInvalid", err)
	}
	if len(r.audit) != 1 {
		t.Errorf("audit entries = %d, want 1 (refused calls write nothing)", len(r.audit))
	}
}
