package migrations_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

// redeemedCoupon inserts a coupon and redeems it at the branch, returning its id.
func redeemedCoupon(ctx context.Context, t *testing.T, pool *pgxpool.Pool, code string, owner, business, customer, branch int64) int64 {
	t.Helper()
	if err := insertCode(ctx, pool, code, insertBenefit(ctx, t, pool, business), business, customer); err != nil {
		t.Fatalf("insert coupon: %v", err)
	}
	var id int64
	if err := pool.QueryRow(ctx,
		`UPDATE benefit_codes SET status = 'redeemed', redeemed_at = now(), redeemed_by = $1, redeemed_branch_id = $2
		 WHERE code = $3 RETURNING id`, owner, branch, code).Scan(&id); err != nil {
		t.Fatalf("redeem coupon: %v", err)
	}
	return id
}

func insertReview(ctx context.Context, pool *pgxpool.Pool, branch, business, coupon, customer int64, rating int, comment any, displayName string) error {
	_, err := pool.Exec(ctx,
		`INSERT INTO reviews (branch_id, business_id, benefit_code_id, customer_id, rating, comment, display_name)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		branch, business, coupon, customer, rating, comment, displayName)
	return err
}

// TestReviewsOnlyExistForARedeemedCouponAtItsBranch pins that the database refuses a
// review without a redeemed coupon, for another branch than the one that redeemed it,
// or a second one for the same coupon.
func TestReviewsOnlyExistForARedeemedCouponAtItsBranch(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	owner := insertOwner(ctx, t, pool, "owner@example.com")
	business, err := insertBusiness(ctx, pool, owner, "studio", "5491112345678", "active")
	if err != nil {
		t.Fatalf("insert business: %v", err)
	}
	branch := insertBranch(ctx, t, pool, business)
	elsewhere := insertBranch(ctx, t, pool, business)
	customer := insertCustomer(ctx, t, pool, "5491112345678")
	redeemed := redeemedCoupon(ctx, t, pool, "AAAAAAAA", owner, business, customer, branch)

	// Before the first review, so the branch mismatch is what gets refused.
	requireSQLState(t, insertReview(ctx, pool, elsewhere, business, redeemed, customer, 5, nil, "Ana G."), foreignKeyViolation)
	if err := insertReview(ctx, pool, branch, business, redeemed, customer, 5, "Great", "Ana G."); err != nil {
		t.Fatalf("a review of a redeemed coupon at its branch must be allowed: %v", err)
	}
	requireSQLState(t, insertReview(ctx, pool, branch, business, redeemed, customer, 4, nil, "Ana G."), uniqueViolation)

	if err := insertCode(ctx, pool, "BBBBBBBB", insertBenefit(ctx, t, pool, business), business, insertCustomer(ctx, t, pool, "5491187654321")); err != nil {
		t.Fatalf("insert pending coupon: %v", err)
	}
	var pending int64
	if err := pool.QueryRow(ctx, `SELECT id FROM benefit_codes WHERE code = 'BBBBBBBB'`).Scan(&pending); err != nil {
		t.Fatalf("read pending coupon: %v", err)
	}
	requireSQLState(t, insertReview(ctx, pool, branch, business, pending, customer, 5, nil, "Ana G."), foreignKeyViolation)
}

// TestReviewShapeIsBounded pins the rating of 1 to 5, the optional comment of up to
// 500 characters, a name to show, and the three moderation states.
func TestReviewShapeIsBounded(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	owner := insertOwner(ctx, t, pool, "owner@example.com")
	business, err := insertBusiness(ctx, pool, owner, "studio", "5491112345678", "active")
	if err != nil {
		t.Fatalf("insert business: %v", err)
	}
	branch := insertBranch(ctx, t, pool, business)
	customer := insertCustomer(ctx, t, pool, "5491112345678")
	coupon := redeemedCoupon(ctx, t, pool, "AAAAAAAA", owner, business, customer, branch)

	for _, bad := range []struct {
		name        string
		rating      int
		comment     any
		displayName string
	}{
		{"rating 0", 0, nil, "Ana G."},
		{"rating 6", 6, nil, "Ana G."},
		{"comment over 500", 5, strings.Repeat("a", 501), "Ana G."},
		{"blank comment", 5, "   ", "Ana G."},
		{"blank name", 5, nil, "  "},
	} {
		requireSQLState(t, insertReview(ctx, pool, branch, business, coupon, customer, bad.rating, bad.comment, bad.displayName), checkViolation)
	}

	if err := insertReview(ctx, pool, branch, business, coupon, customer, 1, strings.Repeat("a", 500), "Cliente de MOVO"); err != nil {
		t.Fatalf("rating 1 with a 500-character comment must be allowed: %v", err)
	}
	var status string
	var showName bool
	if err := pool.QueryRow(ctx, `SELECT status, show_name FROM reviews WHERE benefit_code_id = $1`, coupon).Scan(&status, &showName); err != nil || status != "visible" || !showName {
		t.Errorf("defaults = %q, show_name %v, %v; want visible and true", status, showName, err)
	}
	for _, ok := range []string{"reported", "hidden", "visible"} {
		if _, err := pool.Exec(ctx, `UPDATE reviews SET status = $2 WHERE benefit_code_id = $1`, coupon, ok); err != nil {
			t.Errorf("status %q must be allowed: %v", ok, err)
		}
	}
	_, err = pool.Exec(ctx, `UPDATE reviews SET status = 'deleted' WHERE benefit_code_id = $1`, coupon)
	requireSQLState(t, err, checkViolation)
}

// TestReviewsAreHistory pins that nothing a review points at can be deleted: the owner
// cannot delete reviews and moderation hides them instead.
func TestReviewsAreHistory(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	owner := insertOwner(ctx, t, pool, "owner@example.com")
	business, err := insertBusiness(ctx, pool, owner, "studio", "5491112345678", "active")
	if err != nil {
		t.Fatalf("insert business: %v", err)
	}
	branch := insertBranch(ctx, t, pool, business)
	customer := insertCustomer(ctx, t, pool, "5491112345678")
	coupon := redeemedCoupon(ctx, t, pool, "AAAAAAAA", owner, business, customer, branch)
	if err := insertReview(ctx, pool, branch, business, coupon, customer, 5, nil, "Ana G."); err != nil {
		t.Fatalf("insert review: %v", err)
	}

	for _, del := range []struct {
		stmt string
		id   int64
	}{
		{`DELETE FROM benefit_codes WHERE id = $1`, coupon},
		{`DELETE FROM branches WHERE id = $1`, branch},
		{`DELETE FROM customers WHERE id = $1`, customer},
		{`DELETE FROM businesses WHERE id = $1`, business},
	} {
		_, err := pool.Exec(ctx, del.stmt, del.id)
		requireSQLState(t, err, foreignKeyViolation)
	}
}
