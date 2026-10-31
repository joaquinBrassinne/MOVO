package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/billing"
	"github.com/joaquinBrassinne/MOVO/internal/billing/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/billing/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

type fixture struct {
	pool  *pgxpool.Pool
	repo  *postgres.Subscriptions
	admin int64
	n     int
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	pool := dbtest.New(t)
	f := &fixture{pool: pool, repo: postgres.NewSubscriptions(pool)}
	if err := pool.QueryRow(t.Context(),
		`INSERT INTO users (email, password_hash, role) VALUES ('root@example.com', 'hash', 'super_admin') RETURNING id`).Scan(&f.admin); err != nil {
		t.Fatalf("insert admin: %v", err)
	}
	return f
}

func (f *fixture) business(t *testing.T, status string) int64 {
	t.Helper()
	f.n++
	var owner, business int64
	if err := f.pool.QueryRow(t.Context(),
		`INSERT INTO users (email, password_hash, role) VALUES ($1, 'hash', 'business_admin') RETURNING id`,
		fmt.Sprintf("owner%d@example.com", f.n)).Scan(&owner); err != nil {
		t.Fatalf("insert owner: %v", err)
	}
	if err := f.pool.QueryRow(t.Context(), `
INSERT INTO businesses (owner_user_id, category_id, name, slug, whatsapp_number, status)
VALUES ($1, (SELECT id FROM categories WHERE slug = 'yoga'), 'Studio', $2, '5491112345678', $3) RETURNING id`,
		owner, fmt.Sprintf("studio-%d", f.n), status).Scan(&business); err != nil {
		t.Fatalf("insert business: %v", err)
	}
	return business
}

func sub(t *testing.T, business int64, plan domain.Plan, offsetDays int) domain.Subscription {
	t.Helper()
	s, err := domain.NewSubscription(business, plan, domain.DayOf(time.Now()).AddDays(offsetDays))
	if err != nil {
		t.Fatalf("new subscription: %v", err)
	}
	return s
}

func entry(admin, business int64, action string) billing.AuditEntry {
	return billing.AuditEntry{ActorID: admin, Action: action, BusinessID: business, Metadata: map[string]any{"k": "v"}}
}

func (f *fixture) count(t *testing.T, query string, args ...any) (n int) {
	t.Helper()
	if err := f.pool.QueryRow(t.Context(), query, args...).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

func (f *fixture) status(t *testing.T, business int64) (s string) {
	t.Helper()
	if err := f.pool.QueryRow(t.Context(), `SELECT status FROM businesses WHERE id = $1`, business).Scan(&s); err != nil {
		t.Fatalf("status: %v", err)
	}
	return s
}

func TestConfirmActivatesAPendingBusinessAndStoresTheSubscriptionAndTheAudit(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	ctx := t.Context()
	business := f.business(t, "pending_review")
	want, err := sub(t, business, domain.PlanPro, 30).WithDetails("MP-1", "paid by transfer")
	if err != nil {
		t.Fatalf("details: %v", err)
	}

	if err := f.repo.Confirm(ctx, want, f.admin, entry(f.admin, business, "billing.confirm_payment")); err != nil {
		t.Fatalf("Confirm: %v", err)
	}

	if got := f.status(t, business); got != "active" {
		t.Errorf("business status = %q, want active", got)
	}
	got, err := f.repo.Find(ctx, business)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if got != want {
		t.Errorf("stored = %+v, want %+v", got, want)
	}
	if n := f.count(t, `SELECT count(*) FROM audit_log WHERE entity = 'business' AND entity_id = $1 AND actor_id = $2 AND action = 'billing.confirm_payment'`, business, f.admin); n != 1 {
		t.Errorf("audit entries = %d, want 1", n)
	}
}

func TestConfirmLeavesAnActiveBusinessActiveAndRefusesTheOthersWithoutWritingAnything(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	ctx := t.Context()

	active := f.business(t, "active")
	if err := f.repo.Confirm(ctx, sub(t, active, domain.PlanStarter, 30), f.admin, entry(f.admin, active, "billing.confirm_payment")); err != nil {
		t.Fatalf("confirming an active business: %v", err)
	}
	if got := f.status(t, active); got != "active" {
		t.Errorf("active business status = %q, want active", got)
	}

	for _, status := range []string{"draft", "paused", "deleted"} {
		business := f.business(t, status)
		err := f.repo.Confirm(ctx, sub(t, business, domain.PlanStarter, 30), f.admin, entry(f.admin, business, "billing.confirm_payment"))
		if !errors.Is(err, billing.ErrBusinessNotConfirmable) {
			t.Errorf("%s: error = %v, want ErrBusinessNotConfirmable", status, err)
		}
		if got := f.status(t, business); got != status {
			t.Errorf("%s: status became %q", status, got)
		}
		if n := f.count(t, `SELECT count(*) FROM subscriptions WHERE business_id = $1`, business); n != 0 {
			t.Errorf("%s: %d subscriptions stored, want 0", status, n)
		}
		if n := f.count(t, `SELECT count(*) FROM audit_log WHERE entity_id = $1`, business); n != 0 {
			t.Errorf("%s: %d audit entries stored, want 0", status, n)
		}
	}

	if err := f.repo.Confirm(ctx, sub(t, 999999, domain.PlanStarter, 30), f.admin, entry(f.admin, 999999, "x")); !errors.Is(err, billing.ErrBusinessNotFound) {
		t.Errorf("unknown business error = %v, want ErrBusinessNotFound", err)
	}
}

// A failed confirmation must not leave the business half done: when the subscription
// already exists, the pending status is not flipped and no audit line is written.
func TestConfirmTwiceKeepsTheFirstSubscription(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	ctx := t.Context()
	business := f.business(t, "pending_review")
	if err := f.repo.Confirm(ctx, sub(t, business, domain.PlanPro, 30), f.admin, entry(f.admin, business, "billing.confirm_payment")); err != nil {
		t.Fatalf("first Confirm: %v", err)
	}

	err := f.repo.Confirm(ctx, sub(t, business, domain.PlanStarter, 5), f.admin, entry(f.admin, business, "billing.confirm_payment"))
	if !errors.Is(err, billing.ErrSubscriptionExists) {
		t.Fatalf("second Confirm error = %v, want ErrSubscriptionExists", err)
	}
	got, _ := f.repo.Find(ctx, business)
	if got.Plan != domain.PlanPro {
		t.Errorf("plan = %s, want the first one (pro)", got.Plan)
	}
	if n := f.count(t, `SELECT count(*) FROM audit_log WHERE entity_id = $1`, business); n != 1 {
		t.Errorf("audit entries = %d, want 1", n)
	}
}

func TestOnlyOneOfSeveralSimultaneousConfirmationsWins(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	business := f.business(t, "pending_review")

	const attempts = 8
	results := make(chan error, attempts)
	var wg sync.WaitGroup
	for range attempts {
		wg.Go(func() {
			results <- f.repo.Confirm(context.Background(), sub(t, business, domain.PlanPro, 30), f.admin, entry(f.admin, business, "billing.confirm_payment"))
		})
	}
	wg.Wait()
	close(results)

	var won, exists int
	for err := range results {
		switch {
		case err == nil:
			won++
		case errors.Is(err, billing.ErrSubscriptionExists):
			exists++
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	if won != 1 || exists != attempts-1 {
		t.Errorf("won = %d, exists = %d, want 1 and %d", won, exists, attempts-1)
	}
	if n := f.count(t, `SELECT count(*) FROM audit_log WHERE entity_id = $1`, business); n != 1 {
		t.Errorf("audit entries = %d, want 1", n)
	}
}

func TestUpdateStoresTheChangeAndTheAuditAndRefusesAStaleWrite(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	ctx := t.Context()
	business := f.business(t, "active")
	current := sub(t, business, domain.PlanStarter, 10)
	if err := f.repo.Confirm(ctx, current, f.admin, entry(f.admin, business, "billing.confirm_payment")); err != nil {
		t.Fatalf("Confirm: %v", err)
	}

	paid := current.RegisterPayment(domain.DayOf(time.Now()))
	if err := f.repo.Update(ctx, paid, current, entry(f.admin, business, "billing.register_payment")); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := f.repo.Find(ctx, business)
	if got.PaidUntil != paid.PaidUntil {
		t.Errorf("paid_until = %s, want %s", got.PaidUntil, paid.PaidUntil)
	}

	// A second admin still holding the old state must not overwrite the payment.
	stale := current
	stale.Plan = domain.PlanPro
	if err := f.repo.Update(ctx, stale, current, entry(f.admin, business, "billing.change_plan")); !errors.Is(err, billing.ErrSubscriptionConflict) {
		t.Errorf("stale Update error = %v, want ErrSubscriptionConflict", err)
	}
	if n := f.count(t, `SELECT count(*) FROM audit_log WHERE entity_id = $1 AND action = 'billing.change_plan'`, business); n != 0 {
		t.Errorf("a refused update wrote %d audit entries", n)
	}

	other := f.business(t, "active")
	orphan := sub(t, other, domain.PlanStarter, 10)
	if err := f.repo.Update(ctx, orphan, orphan, entry(f.admin, other, "x")); !errors.Is(err, billing.ErrSubscriptionNotFound) {
		t.Errorf("Update without a subscription error = %v, want ErrSubscriptionNotFound", err)
	}
}

func TestPlanOfIsTheSubscriptionPlanAndStarterWithoutOne(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	ctx := t.Context()
	business := f.business(t, "pending_review")

	if got, err := f.repo.PlanOf(ctx, business); err != nil || got != domain.PlanStarter {
		t.Errorf("PlanOf without a subscription = %q, %v, want starter", got, err)
	}
	if err := f.repo.Confirm(ctx, sub(t, business, domain.PlanPro, 30), f.admin, entry(f.admin, business, "billing.confirm_payment")); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if got, err := f.repo.PlanOf(ctx, business); err != nil || got != domain.PlanPro {
		t.Errorf("PlanOf = %q, %v, want pro", got, err)
	}
}
