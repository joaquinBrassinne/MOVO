package migrations_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/platform/db"
	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

func insertBenefit(ctx context.Context, t *testing.T, pool *pgxpool.Pool, business int64) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO benefits (business_id, type, percent) VALUES ($1, 'percent', 10) RETURNING id`,
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

// TestBenefitShapeFollowsTheType pins the fields each benefit type carries: a percent of
// 1 to 100, a positive amount, or a custom title, each with optional conditions.
func TestBenefitShapeFollowsTheType(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	business, err := insertBusiness(ctx, pool, insertOwner(ctx, t, pool, "owner@example.com"), "studio", "5491112345678", "active")
	if err != nil {
		t.Fatalf("insert business: %v", err)
	}

	for _, tc := range []struct {
		name       string
		kind       string
		percent    any
		amount     any
		title      any
		conditions any
		wantCheck  bool
	}{
		{"percent 1", "percent", 1, nil, nil, nil, false},
		{"percent 37", "percent", 37, nil, nil, nil, false},
		{"percent 100", "percent", 100, nil, nil, nil, false},
		{"percent 0", "percent", 0, nil, nil, nil, true},
		{"percent 101", "percent", 101, nil, nil, nil, true},
		{"percent without a value", "percent", nil, nil, nil, nil, true},
		{"percent with a title", "percent", 10, nil, "Mate", nil, true},
		{"percent with an amount", "percent", 10, 500, nil, nil, true},
		{"percent with conditions", "percent", 10, nil, nil, "Solo lunes", false},
		{"amount", "amount", nil, 2000, nil, nil, false},
		{"amount zero", "amount", nil, 0, nil, nil, true},
		{"amount negative", "amount", nil, -1, nil, nil, true},
		{"amount without a value", "amount", nil, nil, nil, nil, true},
		{"amount with a percent", "amount", 10, 2000, nil, nil, true},
		{"custom", "custom", nil, nil, "Una clase gratis", nil, false},
		{"custom with conditions", "custom", nil, nil, "Una clase gratis", strings.Repeat("a", 120), false},
		{"custom without a title", "custom", nil, nil, nil, nil, true},
		{"custom with a blank title", "custom", nil, nil, "  ", nil, true},
		{"custom with a title of 81", "custom", nil, nil, strings.Repeat("a", 81), nil, true},
		{"custom with a percent", "custom", 10, nil, "Mate", nil, true},
		{"conditions of 121", "custom", nil, nil, "Mate", strings.Repeat("a", 121), true},
		{"blank conditions", "custom", nil, nil, "Mate", "  ", true},
		{"old welcome gift", "welcome_gift", nil, nil, "Mate", nil, true},
		{"unknown type", "two_for_one", nil, nil, "Mate", nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := pool.Exec(ctx,
				`INSERT INTO benefits (business_id, type, percent, amount_ars, title, conditions) VALUES ($1, $2, $3, $4, $5, $6)`,
				business, tc.kind, tc.percent, tc.amount, tc.title, tc.conditions)
			if tc.wantCheck {
				requireSQLState(t, err, checkViolation)
			} else if err != nil {
				t.Errorf("must be allowed: %v", err)
			}
		})
	}

	var paused bool
	if err := pool.QueryRow(ctx, `SELECT admin_paused FROM benefits LIMIT 1`).Scan(&paused); err != nil || paused {
		t.Errorf("admin_paused default = %v, %v; want false", paused, err)
	}
}

// TestBenefitTypesMigrationKeepsTheOldBenefits rolls the migration back, stores benefits
// the old way, and applies it again: percents stay and a welcome gift becomes a custom
// benefit without losing a character of its description.
func TestBenefitTypesMigrationKeepsTheOldBenefits(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	business, err := insertBusiness(ctx, pool, insertOwner(ctx, t, pool, "owner@example.com"), "studio", "5491112345678", "active")
	if err != nil {
		t.Fatalf("insert business: %v", err)
	}
	// Back to the schema of 00014: the quota migration first, then the types one.
	for range 2 {
		if err := db.Migrate(ctx, pool, "down"); err != nil {
			t.Fatalf("roll back: %v", err)
		}
	}

	long := strings.Repeat("a", 80) + strings.Repeat("b", 120)
	for _, old := range []struct {
		kind    string
		percent any
		gift    any
	}{{"percent", 15, nil}, {"welcome_gift", nil, "Una clase gratis"}, {"welcome_gift", nil, long}} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO benefits (business_id, type, percent, gift_description, monthly_limit) VALUES ($1, $2, $3, $4, 5)`,
			business, old.kind, old.percent, old.gift); err != nil {
			t.Fatalf("insert old benefit %v: %v", old.kind, err)
		}
	}
	if err := db.Migrate(ctx, pool, "up"); err != nil {
		t.Fatalf("apply: %v", err)
	}

	type stored struct {
		kind       string
		percent    *int
		title      *string
		conditions *string
		paused     bool
	}
	var got []stored
	rows, err := pool.Query(ctx, `SELECT type, percent, title, conditions, admin_paused FROM benefits ORDER BY id`)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var s stored
		if err := rows.Scan(&s.kind, &s.percent, &s.title, &s.conditions, &s.paused); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got = append(got, s)
	}
	if len(got) != 3 || got[0].kind != "percent" || *got[0].percent != 15 || got[0].title != nil || got[0].paused {
		t.Fatalf("rows = %+v", got)
	}
	if got[1].kind != "custom" || *got[1].title != "Una clase gratis" || got[1].conditions != nil || got[1].percent != nil {
		t.Errorf("short gift = %+v", got[1])
	}
	if got[2].kind != "custom" || *got[2].title != strings.Repeat("a", 80) || got[2].conditions == nil || *got[2].conditions != strings.Repeat("b", 120) {
		t.Errorf("long gift = %+v; want it split into the title and the conditions", got[2])
	}
}

// TestBenefitTypesRollbackRefusesWhatItCannotRepresent pins that going back never changes
// what an owner offers, and that a paused benefit never comes back visible.
func TestBenefitTypesRollbackRefusesWhatItCannotRepresent(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	business, err := insertBusiness(ctx, pool, insertOwner(ctx, t, pool, "owner@example.com"), "studio", "5491112345678", "active")
	if err != nil {
		t.Fatalf("insert business: %v", err)
	}

	if _, err := pool.Exec(ctx, `INSERT INTO benefits (business_id, type, amount_ars) VALUES ($1, 'amount', 2000)`, business); err != nil {
		t.Fatalf("insert amount: %v", err)
	}
	// The first step back is the quota migration, which loses nothing; the second one
	// is the types migration, which cannot represent a fixed amount.
	if err := db.Migrate(ctx, pool, "down"); err != nil {
		t.Fatalf("roll back the quota migration: %v", err)
	}
	if err := db.Migrate(ctx, pool, "down"); err == nil {
		t.Fatal("rolling back with a fixed-amount benefit must fail")
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM benefits WHERE type = 'amount'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("after a refused rollback: %d amount benefits, %v; want it untouched", n, err)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM benefits`); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO benefits (business_id, type, title, conditions, admin_paused, monthly_limit) VALUES ($1, 'custom', 'Mate', 'Solo hoy', true, 5)`, business); err != nil {
		t.Fatalf("insert paused: %v", err)
	}
	if err := db.Migrate(ctx, pool, "down"); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	var kind, gift string
	var active bool
	if err := pool.QueryRow(ctx, `SELECT type, gift_description, active FROM benefits`).Scan(&kind, &gift, &active); err != nil {
		t.Fatalf("read: %v", err)
	}
	if kind != "welcome_gift" || gift != "Mate Solo hoy" || active {
		t.Errorf("after rollback = %q %q active=%v; want the text kept and the paused benefit off", kind, gift, active)
	}
	if err := db.Migrate(ctx, pool, "up"); err != nil {
		t.Fatalf("apply again: %v", err)
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

// TestCouponQuotaLivesOnTheBusiness pins that the monthly coupon limit is a positive
// number or NULL (no limit) on the business, and that rolling back gives each benefit the
// limit of its business, or one no business reaches when it had none.
func TestCouponQuotaLivesOnTheBusiness(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	owner := insertOwner(ctx, t, pool, "owner@example.com")
	business, err := insertBusiness(ctx, pool, owner, "studio", "5491112345678", "active")
	if err != nil {
		t.Fatalf("insert business: %v", err)
	}

	var limit *int
	if err := pool.QueryRow(ctx, `SELECT monthly_coupon_limit FROM businesses WHERE id = $1`, business).Scan(&limit); err != nil || limit != nil {
		t.Fatalf("a new business has limit %v, %v; want NULL (no limit)", limit, err)
	}
	for _, bad := range []int{0, -1} {
		_, err := pool.Exec(ctx, `UPDATE businesses SET monthly_coupon_limit = $2 WHERE id = $1`, business, bad)
		requireSQLState(t, err, checkViolation)
	}
	if _, err := pool.Exec(ctx, `UPDATE businesses SET monthly_coupon_limit = 30 WHERE id = $1`, business); err != nil {
		t.Fatalf("set limit: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE businesses SET monthly_coupon_limit = NULL WHERE id = $1`, business); err != nil {
		t.Fatalf("remove limit: %v", err)
	}
	if _, err := pool.Exec(ctx, `SELECT monthly_limit FROM benefits`); err == nil {
		t.Error("benefits must no longer carry a monthly limit of their own")
	}

	// Rolling back: a business with a limit passes it to its benefits, one without gets the large default.
	other, err := insertBusiness(ctx, pool, insertOwner(ctx, t, pool, "other@example.com"), "other", "5491187654321", "active")
	if err != nil {
		t.Fatalf("insert other business: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE businesses SET monthly_coupon_limit = 30 WHERE id = $1`, other); err != nil {
		t.Fatalf("set limit: %v", err)
	}
	insertBenefit(ctx, t, pool, business)
	insertBenefit(ctx, t, pool, other)
	if err := db.Migrate(ctx, pool, "down"); err != nil {
		t.Fatalf("roll back: %v", err)
	}
	got := map[int64]int{}
	rows, err := pool.Query(ctx, `SELECT business_id, monthly_limit FROM benefits`)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got[id] = n
	}
	if got[business] != 100000 || got[other] != 30 {
		t.Errorf("limits after the rollback = %v, want 100000 without a limit and 30 with one", got)
	}
	if err := db.Migrate(ctx, pool, "up"); err != nil {
		t.Fatalf("apply again: %v", err)
	}
}
