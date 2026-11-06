package migrations_test

import (
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/platform/db"
	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

// couponTerms reads the copy of the benefit terms a coupon carries.
type couponTerms struct {
	kind       string
	percent    *int
	title      *string
	conditions *string
}

// TestCouponTermsAreBackfilledAndFilledForAnyWriter pins the three ways a coupon gets its
// terms: the migration copies them into the coupons that already exist, an insert that
// brings none takes them from the benefit, and an insert that brings its own keeps them.
func TestCouponTermsAreBackfilledAndFilledForAnyWriter(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	business, err := insertBusiness(ctx, pool, insertOwner(ctx, t, pool, "owner@example.com"), "studio", "5491112345678", "active")
	if err != nil {
		t.Fatalf("insert business: %v", err)
	}
	benefit := insertBenefit(ctx, t, pool, business)
	customer := insertCustomer(ctx, t, pool, "5491112345678")

	// A coupon that exists before the migration.
	rollBackTo(ctx, t, pool, 19)
	if err := insertCode(ctx, pool, "ABCDEFGH", benefit, business, customer); err != nil {
		t.Fatalf("insert old coupon: %v", err)
	}
	if err := db.Migrate(ctx, pool, "up"); err != nil {
		t.Fatalf("apply: %v", err)
	}

	read := func(code string) couponTerms {
		t.Helper()
		var c couponTerms
		if err := pool.QueryRow(ctx,
			`SELECT benefit_type, benefit_percent, benefit_title, benefit_conditions FROM benefit_codes WHERE code = $1`, code,
		).Scan(&c.kind, &c.percent, &c.title, &c.conditions); err != nil {
			t.Fatalf("read terms of %s: %v", code, err)
		}
		return c
	}

	if got := read("ABCDEFGH"); got.kind != "percent" || got.percent == nil || *got.percent != 10 || got.title != nil {
		t.Errorf("backfilled coupon = %+v, want the 10%% of its benefit", got)
	}

	// An insert without terms takes them from the benefit as it is now.
	if _, err := pool.Exec(ctx,
		`UPDATE benefits SET type = 'custom', percent = NULL, title = 'Clase gratis', conditions = 'Solo martes' WHERE id = $1`, benefit); err != nil {
		t.Fatalf("edit benefit: %v", err)
	}
	other := insertCustomer(ctx, t, pool, "5491187654321")
	if err := insertCode(ctx, pool, "JKLMNPQR", benefit, business, other); err != nil {
		t.Fatalf("insert coupon without terms: %v", err)
	}
	if got := read("JKLMNPQR"); got.kind != "custom" || got.title == nil || *got.title != "Clase gratis" || got.conditions == nil || *got.conditions != "Solo martes" {
		t.Errorf("filled coupon = %+v, want the terms of the benefit", got)
	}
	if got := read("ABCDEFGH"); got.kind != "percent" || got.percent == nil || *got.percent != 10 {
		t.Errorf("old coupon = %+v, want it untouched by the edit of its benefit", got)
	}

	// An insert that brings its own terms keeps them: this is how Issue pins what the
	// customer was shown, even if the owner edits the benefit in between.
	third := insertCustomer(ctx, t, pool, "5491100000001")
	if _, err := pool.Exec(ctx,
		`INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, expires_at, benefit_type, benefit_percent)
		 VALUES ('STUVWXYZ', $1, $2, $3, sha256(convert_to('STUVWXYZ', 'UTF8')), now() + interval '48 hours', 'percent', 7)`,
		benefit, business, third); err != nil {
		t.Fatalf("insert coupon with terms: %v", err)
	}
	if got := read("STUVWXYZ"); got.kind != "percent" || got.percent == nil || *got.percent != 7 {
		t.Errorf("explicit coupon = %+v, want the 7%% it was inserted with", got)
	}
}

// TestCouponTermsKeepTheShapeOfABenefit pins that the copy follows the same rules as a
// benefit: the fields of each type stay coherent and the conditions stay bounded.
func TestCouponTermsKeepTheShapeOfABenefit(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	business, err := insertBusiness(ctx, pool, insertOwner(ctx, t, pool, "owner@example.com"), "studio", "5491112345678", "active")
	if err != nil {
		t.Fatalf("insert business: %v", err)
	}
	benefit := insertBenefit(ctx, t, pool, business)
	customer := insertCustomer(ctx, t, pool, "5491112345678")

	tests := []struct {
		name  string
		terms string
	}{
		{"percent with a title", `'percent', 10, NULL, 'Gratis', NULL`},
		{"percent out of range", `'percent', 101, NULL, NULL, NULL`},
		{"amount without amount", `'amount', NULL, NULL, NULL, NULL`},
		{"custom without title", `'custom', NULL, NULL, NULL, NULL`},
		{"unknown type", `'gift', NULL, NULL, 'x', NULL`},
		{"blank conditions", `'percent', 10, NULL, NULL, '  '`},
	}
	for i, tt := range tests {
		code := "AAAAAAA" + string(rune('A'+i))
		_, err := pool.Exec(ctx,
			`INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, expires_at,
			        benefit_type, benefit_percent, benefit_amount_ars, benefit_title, benefit_conditions)
			 VALUES ($1, $2, $3, $4, sha256(convert_to($1::text, 'UTF8')), now() + interval '48 hours', `+tt.terms+`)`,
			code, benefit, business, customer)
		if err == nil {
			t.Errorf("%s: the database accepted terms that no benefit could have", tt.name)
			continue
		}
		requireSQLState(t, err, "23514")
	}
}
