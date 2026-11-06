package migrations_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/billing/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

// argentinaToday is the calendar day the view and the domain both count in.
const argentinaToday = `(now() AT TIME ZONE 'America/Argentina/Buenos_Aires')::date`

// paidUntil gives the business a subscription that ends offsetDays from today
// (negative: already ended).
func paidUntil(ctx context.Context, t *testing.T, pool *pgxpool.Pool, business, admin int64, offsetDays int) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
		INSERT INTO subscriptions (business_id, plan, paid_until, confirmed_by)
		VALUES ($1, 'starter', `+argentinaToday+` + $3::int, $2)`, business, admin, offsetDays); err != nil {
		t.Fatalf("insert subscription: %v", err)
	}
}

// TestVisibleBranchesFollowThePayment pins SDD 4.1: an active business is public when it
// is a pilot, or when today is on or before paid_until plus the grace days. paid_until
// itself and the last day of grace are still visible; the day after is not.
func TestVisibleBranchesFollowThePayment(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()

	for i, tc := range []struct {
		name  string
		pilot bool
		paid  *int // days from today to paid_until; nil = no subscription
		want  bool
	}{
		{"paid up", false, ptr(20), true},
		{"paid until today", false, ptr(0), true},
		{"in grace, first day", false, ptr(-1), true},
		{"in grace, last day", false, ptr(-domain.GraceDays), true},
		{"expired, first day", false, ptr(-domain.GraceDays - 1), false},
		{"long expired", false, ptr(-90), false},
		{"no subscription", false, nil, false},
		{"pilot without subscription", true, nil, true},
		{"pilot with an expired subscription", true, ptr(-90), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			owner := insertOwner(ctx, t, pool, fmt.Sprintf("owner%d@example.com", i))
			business, err := insertBusiness(ctx, pool, owner, fmt.Sprintf("studio-%d", i), "5491112345678", "active")
			if err != nil {
				t.Fatalf("insert business: %v", err)
			}
			if _, err := pool.Exec(ctx, `UPDATE businesses SET pilot = $2 WHERE id = $1`, business, tc.pilot); err != nil {
				t.Fatalf("set pilot: %v", err)
			}
			if tc.paid != nil {
				paidUntil(ctx, t, pool, business, owner, *tc.paid)
			}
			branch := insertBranch(ctx, t, pool, business)

			var got bool
			if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM visible_branches WHERE branch_id = $1)`, branch).Scan(&got); err != nil {
				t.Fatalf("query: %v", err)
			}
			if got != tc.want {
				t.Errorf("visible = %t, want %t", got, tc.want)
			}
		})
	}
}

// TestGraceDaysAgreeWithTheDomain pins that the 7 written in the view is the number the
// domain uses: a view cannot read configuration, so this is the only guard against the two
// drifting apart.
func TestGraceDaysAgreeWithTheDomain(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	var def string
	if err := pool.QueryRow(t.Context(), `SELECT pg_get_viewdef('visible_branches'::regclass)`).Scan(&def); err != nil {
		t.Fatalf("view definition: %v", err)
	}
	if want := fmt.Sprintf("paid_until + %d", domain.GraceDays); !strings.Contains(def, want) {
		t.Errorf("view definition does not contain %q:\n%s", want, def)
	}
}

// TestVisibleBranchesFeatureOnlyPayingProBusinesses pins featured: a paying Pro business,
// not a pilot, not Starter.
func TestVisibleBranchesFeatureOnlyPayingProBusinesses(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()

	for i, tc := range []struct {
		name  string
		pilot bool
		plan  string
		want  bool
	}{
		{"pro", false, "pro", true},
		{"starter", false, "starter", false},
		{"pilot", true, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			owner := insertOwner(ctx, t, pool, fmt.Sprintf("owner%d@example.com", i))
			business, err := insertBusiness(ctx, pool, owner, fmt.Sprintf("studio-%d", i), "5491112345678", "active")
			if err != nil {
				t.Fatalf("insert business: %v", err)
			}
			if _, err := pool.Exec(ctx, `UPDATE businesses SET pilot = $2 WHERE id = $1`, business, tc.pilot); err != nil {
				t.Fatalf("set pilot: %v", err)
			}
			if tc.plan != "" {
				if _, err := pool.Exec(ctx, `
					INSERT INTO subscriptions (business_id, plan, paid_until, confirmed_by)
					VALUES ($1, $2, CURRENT_DATE + 30, $3)`, business, tc.plan, owner); err != nil {
					t.Fatalf("insert subscription: %v", err)
				}
			}
			branch := insertBranch(ctx, t, pool, business)

			var got bool
			if err := pool.QueryRow(ctx, `SELECT featured FROM visible_branches WHERE branch_id = $1`, branch).Scan(&got); err != nil {
				t.Fatalf("query: %v", err)
			}
			if got != tc.want {
				t.Errorf("featured = %t, want %t", got, tc.want)
			}
		})
	}
}

func ptr(n int) *int { return &n }
