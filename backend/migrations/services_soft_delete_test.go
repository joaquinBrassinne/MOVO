package migrations_test

import (
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

// TestDeletingAServiceWithMetricsIsRefused pins the rule that a service is never
// physically deleted once a counter points at it: the app marks it deleted instead.
func TestDeletingAServiceWithMetricsIsRefused(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	business, err := insertBusiness(ctx, pool, insertOwner(ctx, t, pool, "owner@example.com"), "studio", "5491112345678", "active")
	if err != nil {
		t.Fatalf("insert business: %v", err)
	}
	branch := insertBranch(ctx, t, pool, business)

	var counted, empty int64
	for _, id := range []*int64{&counted, &empty} {
		if err := pool.QueryRow(ctx, `INSERT INTO services (business_id, name) VALUES ($1, 'Yoga') RETURNING id`, business).Scan(id); err != nil {
			t.Fatalf("insert service: %v", err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO metric_daily (business_id, branch_id, service_id, date, metric, count) VALUES ($1, $2, $3, '2026-03-01', 'service_view', 5)`, business, branch, counted); err != nil {
		t.Fatalf("insert counter: %v", err)
	}

	_, err = pool.Exec(ctx, `DELETE FROM services WHERE id = $1`, counted)
	requireSQLState(t, err, foreignKeyViolation)
	if _, err := pool.Exec(ctx, `DELETE FROM services WHERE id = $1`, empty); err != nil {
		t.Errorf("a service with no history can still be deleted: %v", err)
	}
}
