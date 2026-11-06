package postgres_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/billing"
	"github.com/joaquinBrassinne/MOVO/internal/billing/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/billing/domain"
)

func (f *fixture) addBranches(t *testing.T, business int64, n int) []int64 {
	t.Helper()
	var ids []int64
	for i := range n {
		var id int64
		if err := f.pool.QueryRow(t.Context(), `
INSERT INTO branches (business_id, name, address, city, location)
VALUES ($1, $2, 'Calle 1', 'Cordoba', ST_SetSRID(ST_MakePoint(-64.18, -31.42), 4326)::geography) RETURNING id`,
			business, fmt.Sprintf("Branch %d", i+1)).Scan(&id); err != nil {
			t.Fatalf("insert branch: %v", err)
		}
		ids = append(ids, id)
	}
	return ids
}

func (f *fixture) addServices(t *testing.T, business int64, n int) []int64 {
	t.Helper()
	var ids []int64
	for i := range n {
		var id int64
		if err := f.pool.QueryRow(t.Context(), `
INSERT INTO services (business_id, name, description, price) VALUES ($1, $2, 'x', 1000) RETURNING id`,
			business, fmt.Sprintf("Service %d", i+1)).Scan(&id); err != nil {
			t.Fatalf("insert service: %v", err)
		}
		ids = append(ids, id)
	}
	return ids
}

func (f *fixture) active(t *testing.T, table string, business int64) (ids []int64) {
	t.Helper()
	rows, err := f.pool.Query(t.Context(), `SELECT id FROM `+table+` WHERE business_id = $1 AND active ORDER BY id`, business)
	if err != nil {
		t.Fatalf("active %s: %v", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		ids = append(ids, id)
	}
	return ids
}

func (f *fixture) owner(t *testing.T, business int64) (id int64) {
	t.Helper()
	if err := f.pool.QueryRow(t.Context(), `SELECT owner_user_id FROM businesses WHERE id = $1`, business).Scan(&id); err != nil {
		t.Fatalf("owner: %v", err)
	}
	return id
}

// starter is the limits of SDD 4.2: 2 branches and 3 active services; pro has 4 and unlimited.
func limits(t *testing.T, branches, services int) domain.Entitlements {
	t.Helper()
	e, err := domain.NewEntitlements(branches, services)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func equal(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestChangePlanSwitchesOffWhatTheNewPlanDoesNotFitKeepingTheOldest(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	ctx := t.Context()
	business := f.business(t, "active")
	branches := f.addBranches(t, business, 4)
	services := f.addServices(t, business, 5)
	pro := sub(t, business, domain.PlanPro, 30)
	if err := f.repo.Confirm(ctx, pro, f.admin, entry(f.admin, business, "billing.confirm_payment")); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	starter, _ := pro.ChangePlan(domain.PlanStarter)

	down, err := f.repo.ChangePlan(ctx, starter, pro, limits(t, 2, 3), entry(f.admin, business, "billing.change_plan"))
	if err != nil {
		t.Fatalf("ChangePlan: %v", err)
	}

	// The oldest stay active; the newest are switched off; nothing is deleted.
	if got := f.active(t, "branches", business); !equal(got, branches[:2]) {
		t.Errorf("active branches = %v, want the oldest two %v", got, branches[:2])
	}
	if got := f.active(t, "services", business); !equal(got, services[:3]) {
		t.Errorf("active services = %v, want the oldest three %v", got, services[:3])
	}
	if n := f.count(t, `SELECT count(*) FROM branches WHERE business_id = $1`, business); n != 4 {
		t.Errorf("branches stored = %d, want 4 (nothing is deleted)", n)
	}
	if n := f.count(t, `SELECT count(*) FROM services WHERE business_id = $1`, business); n != 5 {
		t.Errorf("services stored = %d, want 5 (nothing is deleted)", n)
	}
	if len(down.Branches) != 2 || down.Branches[0].ID != branches[2] || down.Branches[0].Name != "Branch 3" || len(down.Services) != 2 {
		t.Errorf("downgrade = %+v", down)
	}

	// The audit line says which ones, and the owner has one unread alert with the details.
	var audited int
	if err := f.pool.QueryRow(ctx, `
SELECT count(*) FROM audit_log
WHERE entity_id = $1 AND action = 'billing.change_plan'
  AND jsonb_array_length(metadata->'branches_deactivated') = 2 AND jsonb_array_length(metadata->'services_deactivated') = 2`, business).Scan(&audited); err != nil || audited != 1 {
		t.Errorf("audit lines with the switched-off ids = %d (%v), want 1", audited, err)
	}
	alerts, err := postgres.NewAlerts(f.pool).Unread(ctx, f.owner(t, business))
	if err != nil || len(alerts) != 1 {
		t.Fatalf("alerts = %v, %v, want one", alerts, err)
	}
	if alerts[0].Type != "plan_downgrade" || alerts[0].Payload["plan_from"] != "pro" || alerts[0].Payload["plan_to"] != "starter" {
		t.Errorf("alert = %+v", alerts[0])
	}
	if got := alerts[0].Payload["branches"].([]any); len(got) != 2 || got[0].(map[string]any)["name"] != "Branch 3" {
		t.Errorf("alert branches = %v, want the two switched off by name", got)
	}
}

func TestChangePlanThatFitsLeavesEverythingAndWritesNoAlert(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	ctx := t.Context()
	business := f.business(t, "active")
	f.addBranches(t, business, 2)
	starter := sub(t, business, domain.PlanStarter, 30)
	if err := f.repo.Confirm(ctx, starter, f.admin, entry(f.admin, business, "billing.confirm_payment")); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	pro, _ := starter.ChangePlan(domain.PlanPro)

	down, err := f.repo.ChangePlan(ctx, pro, starter, limits(t, 4, 0), entry(f.admin, business, "billing.change_plan"))
	if err != nil || !down.Empty() {
		t.Fatalf("upgrade = %+v, %v, want nothing switched off", down, err)
	}
	if n := f.count(t, `SELECT count(*) FROM alerts WHERE business_id = $1`, business); n != 0 {
		t.Errorf("alerts = %d, want 0", n)
	}
	if got, _ := f.repo.PlanOf(ctx, business); got != domain.PlanPro {
		t.Errorf("plan = %s, want pro", got)
	}
}

// A pilot business lives on the Pro limits whatever its subscription says, so a downgrade
// of the subscription must not shrink it.
func TestChangePlanDoesNotShrinkAPilotBusiness(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	ctx := t.Context()
	business := f.business(t, "active")
	branches := f.addBranches(t, business, 4)
	if _, err := f.pool.Exec(ctx, `UPDATE businesses SET pilot = true WHERE id = $1`, business); err != nil {
		t.Fatalf("mark pilot: %v", err)
	}
	pro := sub(t, business, domain.PlanPro, 30)
	if err := f.repo.Confirm(ctx, pro, f.admin, entry(f.admin, business, "billing.confirm_payment")); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	starter, _ := pro.ChangePlan(domain.PlanStarter)

	down, err := f.repo.ChangePlan(ctx, starter, pro, limits(t, 2, 3), entry(f.admin, business, "billing.change_plan"))
	if err != nil || !down.Empty() {
		t.Fatalf("downgrade of a pilot = %+v, %v, want nothing switched off", down, err)
	}
	if got := f.active(t, "branches", business); !equal(got, branches) {
		t.Errorf("active branches = %v, want all %v", got, branches)
	}
}

// A stale change must leave every branch and service as it was: the limits are applied in
// the same transaction as the plan, so a refused plan switches nothing off.
func TestStaleChangePlanSwitchesNothingOff(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	ctx := t.Context()
	business := f.business(t, "active")
	branches := f.addBranches(t, business, 4)
	pro := sub(t, business, domain.PlanPro, 30)
	if err := f.repo.Confirm(ctx, pro, f.admin, entry(f.admin, business, "billing.confirm_payment")); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	// Someone else extends the payment first.
	paid := pro.RegisterPayment(domain.DayOf(pro.PaidUntil.Time()))
	if err := f.repo.Update(ctx, paid, pro, entry(f.admin, business, "billing.register_payment")); err != nil {
		t.Fatalf("Update: %v", err)
	}
	starter, _ := pro.ChangePlan(domain.PlanStarter)

	if _, err := f.repo.ChangePlan(ctx, starter, pro, limits(t, 2, 3), entry(f.admin, business, "billing.change_plan")); !errors.Is(err, billing.ErrSubscriptionConflict) {
		t.Fatalf("error = %v, want ErrSubscriptionConflict", err)
	}
	if got := f.active(t, "branches", business); !equal(got, branches) {
		t.Errorf("active branches = %v, want all %v", got, branches)
	}
	if n := f.count(t, `SELECT count(*) FROM alerts WHERE business_id = $1`, business); n != 0 {
		t.Errorf("alerts = %d, want 0", n)
	}
}

func TestOwnersReadAndDismissOnlyTheirOwnAlerts(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	ctx := t.Context()
	mine, other := f.business(t, "active"), f.business(t, "active")
	alerts := postgres.NewAlerts(f.pool)
	var mineID, otherID int64
	for business, dst := range map[int64]*int64{mine: &mineID, other: &otherID} {
		if err := f.pool.QueryRow(ctx, `INSERT INTO alerts (business_id, type) VALUES ($1, 'plan_downgrade') RETURNING id`, business).Scan(dst); err != nil {
			t.Fatalf("insert alert: %v", err)
		}
	}
	owner := f.owner(t, mine)

	got, err := alerts.Unread(ctx, owner)
	if err != nil || len(got) != 1 || got[0].ID != mineID {
		t.Fatalf("unread = %+v, %v, want only mine", got, err)
	}
	if err := alerts.MarkRead(ctx, owner, otherID); !errors.Is(err, billing.ErrAlertNotFound) {
		t.Errorf("dismissing another business's alert = %v, want ErrAlertNotFound", err)
	}
	if n := f.count(t, `SELECT count(*) FROM alerts WHERE id = $1 AND read_at IS NOT NULL`, otherID); n != 0 {
		t.Error("another owner's alert was marked read")
	}
	if err := alerts.MarkRead(ctx, owner, mineID); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	if err := alerts.MarkRead(ctx, owner, mineID); err != nil {
		t.Errorf("dismissing twice = %v, want no error", err)
	}
	if got, _ := alerts.Unread(ctx, owner); len(got) != 0 {
		t.Errorf("unread after dismissing = %+v, want none", got)
	}
	if got, err := alerts.Unread(ctx, f.admin); err != nil || len(got) != 0 {
		t.Errorf("an owner with no business = %+v, %v, want none", got, err)
	}
}
