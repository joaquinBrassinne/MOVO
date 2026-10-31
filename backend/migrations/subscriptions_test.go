package migrations_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

func insertSubscription(ctx context.Context, pool *pgxpool.Pool, business, admin int64, plan string, ref, notes any) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO subscriptions (business_id, plan, paid_until, mp_subscription_ref, confirmed_by, notes)
		VALUES ($1, $2, CURRENT_DATE + 30, $3, $4, $5)`, business, plan, ref, admin, notes)
	return err
}

// TestSubscriptionsRefuseInvalidRows pins what the database itself rejects: an unknown
// plan, a blank or oversized reference or note, and a second subscription for the same
// business. The plan list matches the PlanCatalog (starter, pro).
func TestSubscriptionsRefuseInvalidRows(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	admin := insertOwner(ctx, t, pool, "admin@example.com")
	business, err := insertBusiness(ctx, pool, insertOwner(ctx, t, pool, "owner@example.com"), "studio", "5491112345678", "active")
	if err != nil {
		t.Fatalf("insert business: %v", err)
	}

	for _, tc := range []struct {
		name  string
		plan  string
		ref   any
		notes any
	}{
		{"unknown plan", "gold", nil, nil},
		{"blank reference", "pro", "  ", nil},
		{"oversized reference", "pro", strings.Repeat("a", 101), nil},
		{"blank notes", "pro", nil, " "},
		{"oversized notes", "pro", nil, strings.Repeat("a", 501)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := insertSubscription(ctx, pool, business, admin, tc.plan, tc.ref, tc.notes); err == nil {
				t.Error("insert succeeded, want a constraint error")
			}
		})
	}

	if err := insertSubscription(ctx, pool, business, admin, "starter", "ref-1", "first payment"); err != nil {
		t.Fatalf("valid insert: %v", err)
	}
	if err := insertSubscription(ctx, pool, business, admin, "pro", nil, nil); err == nil {
		t.Error("second subscription for the same business succeeded, want a unique violation")
	}
}
