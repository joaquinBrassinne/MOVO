package migrations_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

func insertBenefit(ctx context.Context, t *testing.T, pool *pgxpool.Pool, business int64) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO benefits (business_id, type, percent, monthly_limit) VALUES ($1, 'percent', 10, 50) RETURNING id`,
		business).Scan(&id); err != nil {
		t.Fatalf("insert benefit: %v", err)
	}
	return id
}

func insertCustomer(ctx context.Context, t *testing.T, pool *pgxpool.Pool, phone string) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(ctx, `INSERT INTO customers (name, phone) VALUES ('Ana', $1) RETURNING id`, phone).Scan(&id); err != nil {
		t.Fatalf("insert customer: %v", err)
	}
	return id
}

// insertCode inserts a pending coupon; the token hash is derived from the code so each
// call gets a distinct 32-byte value.
func insertCode(ctx context.Context, pool *pgxpool.Pool, code string, benefit, business, customer int64) error {
	_, err := pool.Exec(ctx,
		`INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, expires_at)
		 VALUES ($1, $2, $3, $4, sha256(convert_to($1::text, 'UTF8')), now() + interval '48 hours')`,
		code, benefit, business, customer)
	return err
}

// TestBenefitShapeFollowsTheType pins that a percent benefit carries 10/15/20 and no
// gift, and a welcome gift carries a description and no percent.
func TestBenefitShapeFollowsTheType(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	business, err := insertBusiness(ctx, pool, insertOwner(ctx, t, pool, "owner@example.com"), "studio", "5491112345678", "active")
	if err != nil {
		t.Fatalf("insert business: %v", err)
	}

	for _, tc := range []struct {
		name      string
		kind      string
		percent   any
		gift      any
		limit     int
		wantCheck bool
	}{
		{"percent 10", "percent", 10, nil, 5, false},
		{"percent 15", "percent", 15, nil, 5, false},
		{"percent 20", "percent", 20, nil, 5, false},
		{"percent 25", "percent", 25, nil, 5, true},
		{"percent without a value", "percent", nil, nil, 5, true},
		{"percent with a gift", "percent", 10, "Mate", 5, true},
		{"gift with a description", "welcome_gift", nil, "Una clase gratis", 5, false},
		{"gift without a description", "welcome_gift", nil, nil, 5, true},
		{"gift with a blank description", "welcome_gift", nil, "  ", 5, true},
		{"gift with a percent", "welcome_gift", 10, "Mate", 5, true},
		{"unknown type", "two_for_one", nil, "Mate", 5, true},
		{"zero monthly limit", "percent", 10, nil, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := pool.Exec(ctx,
				`INSERT INTO benefits (business_id, type, percent, gift_description, monthly_limit) VALUES ($1, $2, $3, $4, $5)`,
				business, tc.kind, tc.percent, tc.gift, tc.limit)
			if tc.wantCheck {
				requireSQLState(t, err, checkViolation)
			} else if err != nil {
				t.Errorf("must be allowed: %v", err)
			}
		})
	}
}

// TestCustomerPhoneIsNormalizedAndUnique pins that the contact is stored as digits with
// country code and that one phone is one customer.
func TestCustomerPhoneIsNormalizedAndUnique(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()

	for _, phone := range []string{"+5491112345678", "011 1234 5678", "0123456789", "123"} {
		_, err := pool.Exec(ctx, `INSERT INTO customers (name, phone) VALUES ('Ana', $1)`, phone)
		requireSQLState(t, err, checkViolation)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO customers (name, phone) VALUES ('  ', '5491112345678')`); err == nil {
		t.Error("a blank name must be refused")
	}
	insertCustomer(ctx, t, pool, "5491112345678")
	_, err := pool.Exec(ctx, `INSERT INTO customers (name, phone) VALUES ('Beto', '5491112345678')`)
	requireSQLState(t, err, uniqueViolation)
}

// TestOnlyOnePendingCouponPerCustomerAndBusiness pins the partial unique index: a
// second pending coupon is refused, and redeeming or expiring the first frees the slot.
func TestOnlyOnePendingCouponPerCustomerAndBusiness(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	owner := insertOwner(ctx, t, pool, "owner@example.com")
	business, err := insertBusiness(ctx, pool, owner, "studio", "5491112345678", "active")
	if err != nil {
		t.Fatalf("insert business: %v", err)
	}
	other, err := insertBusiness(ctx, pool, insertOwner(ctx, t, pool, "other@example.com"), "other", "5491112345679", "active")
	if err != nil {
		t.Fatalf("insert other business: %v", err)
	}
	branch := insertBranch(ctx, t, pool, business)
	benefit := insertBenefit(ctx, t, pool, business)
	otherBenefit := insertBenefit(ctx, t, pool, other)
	customer := insertCustomer(ctx, t, pool, "5491112345678")
	someoneElse := insertCustomer(ctx, t, pool, "5491187654321")

	if err := insertCode(ctx, pool, "AAAAAAAA", benefit, business, customer); err != nil {
		t.Fatalf("first coupon: %v", err)
	}
	requireSQLState(t, insertCode(ctx, pool, "BBBBBBBB", benefit, business, customer), uniqueViolation)
	if err := insertCode(ctx, pool, "CCCCCCCC", benefit, business, someoneElse); err != nil {
		t.Errorf("another customer must get a coupon at the same business: %v", err)
	}
	if err := insertCode(ctx, pool, "DDDDDDDD", otherBenefit, other, customer); err != nil {
		t.Errorf("the same customer must get a coupon at another business: %v", err)
	}

	if _, err := pool.Exec(ctx,
		`UPDATE benefit_codes SET status = 'redeemed', redeemed_at = now(), redeemed_by = $1, redeemed_branch_id = $2 WHERE code = 'AAAAAAAA'`,
		owner, branch); err != nil {
		t.Fatalf("redeem: %v", err)
	}
	if err := insertCode(ctx, pool, "BBBBBBBB", benefit, business, customer); err != nil {
		t.Errorf("a redeemed coupon must free the slot: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE benefit_codes SET status = 'expired' WHERE code = 'BBBBBBBB'`); err != nil {
		t.Fatalf("expire: %v", err)
	}
	if err := insertCode(ctx, pool, "EEEEEEEE", benefit, business, customer); err != nil {
		t.Errorf("an expired coupon must free the slot: %v", err)
	}
}

// TestBenefitCodeKeepsItsRulesInTheDatabase pins the code alphabet, the unique code and
// token, the redeemed shape and that a coupon cannot cross businesses.
func TestBenefitCodeKeepsItsRulesInTheDatabase(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	owner := insertOwner(ctx, t, pool, "owner@example.com")
	business, err := insertBusiness(ctx, pool, owner, "studio", "5491112345678", "active")
	if err != nil {
		t.Fatalf("insert business: %v", err)
	}
	other, err := insertBusiness(ctx, pool, insertOwner(ctx, t, pool, "other@example.com"), "other", "5491112345679", "active")
	if err != nil {
		t.Fatalf("insert other business: %v", err)
	}
	branch := insertBranch(ctx, t, pool, business)
	otherBranch := insertBranch(ctx, t, pool, other)
	benefit := insertBenefit(ctx, t, pool, business)
	customer := insertCustomer(ctx, t, pool, "5491112345678")
	buyer := insertCustomer(ctx, t, pool, "5491187654321")

	for _, code := range []string{"abcdefgh", "ABCDEFG", "ABCDEFGHJ", "ABCDEFG0", "ABCDEFGO", "ABCDEFG1", "ABCDEFGI"} {
		requireSQLState(t, insertCode(ctx, pool, code, benefit, business, customer), checkViolation)
	}
	if err := insertCode(ctx, pool, "ABCDEFGH", benefit, business, customer); err != nil {
		t.Fatalf("valid code: %v", err)
	}
	requireSQLState(t, insertCode(ctx, pool, "ABCDEFGH", benefit, business, buyer), uniqueViolation)

	_, err = pool.Exec(ctx,
		`INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, expires_at)
		 VALUES ('JKLMNPQR', $1, $2, $3, sha256(convert_to('ABCDEFGH', 'UTF8')), now())`, benefit, business, buyer)
	requireSQLState(t, err, uniqueViolation)
	_, err = pool.Exec(ctx,
		`INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, expires_at)
		 VALUES ('JKLMNPQS', $1, $2, $3, '\x00', now())`, benefit, business, buyer)
	requireSQLState(t, err, checkViolation)

	// The benefit must belong to the coupon's business.
	requireSQLState(t, insertCode(ctx, pool, "STUVWXYZ", benefit, other, buyer), foreignKeyViolation)

	redeem := func(by, branchID any) error {
		_, err := pool.Exec(ctx,
			`UPDATE benefit_codes SET status = 'redeemed', redeemed_at = now(), redeemed_by = $1, redeemed_branch_id = $2 WHERE code = 'ABCDEFGH'`,
			by, branchID)
		return err
	}
	requireSQLState(t, redeem(nil, branch), checkViolation)
	requireSQLState(t, redeem(owner, nil), checkViolation)
	requireSQLState(t, redeem(owner, otherBranch), foreignKeyViolation)
	if err := redeem(owner, branch); err != nil {
		t.Errorf("a complete redemption must be allowed: %v", err)
	}
	// A pending coupon cannot carry redemption data.
	_, err = pool.Exec(ctx, `UPDATE benefit_codes SET status = 'pending' WHERE code = 'ABCDEFGH'`)
	requireSQLState(t, err, checkViolation)
}

// TestCouponsAreHistory pins that a benefit or a business with coupons cannot be deleted.
func TestCouponsAreHistory(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	business, err := insertBusiness(ctx, pool, insertOwner(ctx, t, pool, "owner@example.com"), "studio", "5491112345678", "active")
	if err != nil {
		t.Fatalf("insert business: %v", err)
	}
	benefit := insertBenefit(ctx, t, pool, business)
	if err := insertCode(ctx, pool, "ABCDEFGH", benefit, business, insertCustomer(ctx, t, pool, "5491112345678")); err != nil {
		t.Fatalf("insert coupon: %v", err)
	}

	_, err = pool.Exec(ctx, `DELETE FROM benefits WHERE id = $1`, benefit)
	requireSQLState(t, err, foreignKeyViolation)
	_, err = pool.Exec(ctx, `DELETE FROM businesses WHERE id = $1`, business)
	requireSQLState(t, err, foreignKeyViolation)
}
