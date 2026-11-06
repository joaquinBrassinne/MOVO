package postgres_test

import (
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/adapters/postgres"
)

// statusCoupon inserts a coupon of the given status created at the instant given. Pending ones
// expire at expiresAt; redeemed ones are redeemed at the shop's branch.
func (f *conversionFixture) statusCoupon(t *testing.T, benefit int64, status string, createdAt, expiresAt time.Time, customer int64) {
	t.Helper()
	f.n++
	code := raceCode(f.n)
	hash := make([]byte, 32)
	copy(hash, code)
	var err error
	if status == "redeemed" {
		_, err = f.pool.Exec(t.Context(), `
INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, status, expires_at, created_at, redeemed_at, redeemed_by, redeemed_branch_id)
VALUES ($1, $2, $3, $4, $5, 'redeemed', $6, $7, $7, $8, $9)`, code, benefit, f.business, customer, hash, expiresAt, createdAt, f.owner, f.branch)
	} else {
		_, err = f.pool.Exec(t.Context(), `
INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, status, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, code, benefit, f.business, customer, hash, status, expiresAt, createdAt)
	}
	if err != nil {
		t.Fatalf("insert %s coupon: %v", status, err)
	}
}

// TestBenefitPerformanceCountsPendingPastTheirTimeAsExpiredWithoutAnyBackgroundTask checks the
// three ways a coupon ends and the cohort window.
func TestBenefitPerformanceCountsPendingPastTheirTimeAsExpiredWithoutAnyBackgroundTask(t *testing.T) {
	t.Parallel()

	f := newConversionFixture(t, "studio")
	var benefit int64
	if err := f.pool.QueryRow(t.Context(), `SELECT id FROM benefits WHERE business_id = $1`, f.business).Scan(&benefit); err != nil {
		t.Fatal(err)
	}
	var second int64
	if err := f.pool.QueryRow(t.Context(), `INSERT INTO customers (name, phone) VALUES ('Beto', '5491100000002') RETURNING id`).Scan(&second); err != nil {
		t.Fatal(err)
	}
	var third int64
	if err := f.pool.QueryRow(t.Context(), `INSERT INTO customers (name, phone) VALUES ('Cata', '5491100000003') RETURNING id`).Scan(&third); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2028, 3, 15, 12, 0, 0, 0, argentina)
	current := periodEnding(now, 7)
	previous := periodEnding(time.Date(2028, 3, 8, 0, 0, 0, 0, argentina), 7)
	inside := time.Date(2028, 3, 13, 10, 0, 0, 0, argentina)

	f.statusCoupon(t, benefit, "redeemed", inside, inside.Add(48*time.Hour), f.customer)
	f.statusCoupon(t, benefit, "expired", inside, inside.Add(48*time.Hour), f.customer)               // marked by the worker
	f.statusCoupon(t, benefit, "pending", inside, now.Add(-time.Minute), second)                      // never marked, but past its time
	f.statusCoupon(t, benefit, "pending", inside.Add(24*time.Hour), now.Add(time.Hour), third)        // still valid
	f.statusCoupon(t, benefit, "expired", time.Date(2028, 3, 5, 10, 0, 0, 0, argentina), now, second) // previous period
	f.statusCoupon(t, benefit, "expired", time.Date(2028, 3, 16, 0, 0, 0, 0, argentina), now, second) // after the period

	got, err := postgres.NewCoupons(f.pool).BenefitPerformance(t.Context(), f.business, current, previous, now)
	if err != nil {
		t.Fatal(err)
	}
	if want := []benefits.BenefitCounts{{BenefitID: benefit, Generated: 4, Redeemed: 1, Expired: 2}}; len(got.Current) != 1 || got.Current[0] != want[0] {
		t.Errorf("current = %+v, want %+v (one pending is still valid)", got.Current, want)
	}
	if want := (benefits.BenefitCounts{BenefitID: benefit, Generated: 1, Expired: 1}); len(got.Previous) != 1 || got.Previous[0] != want {
		t.Errorf("previous = %+v, want %+v", got.Previous, want)
	}
}

func TestBenefitPerformanceKeepsEachBenefitAndEachBusinessApart(t *testing.T) {
	t.Parallel()

	f := newConversionFixture(t, "studio")
	var first int64
	if err := f.pool.QueryRow(t.Context(), `SELECT id FROM benefits WHERE business_id = $1`, f.business).Scan(&first); err != nil {
		t.Fatal(err)
	}
	var second int64
	if err := f.pool.QueryRow(t.Context(), `INSERT INTO benefits (business_id, type, percent) VALUES ($1, 'percent', 20) RETURNING id`, f.business).Scan(&second); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2028, 3, 15, 12, 0, 0, 0, argentina)
	inside := time.Date(2028, 3, 13, 10, 0, 0, 0, argentina)
	f.statusCoupon(t, first, "redeemed", inside, inside.Add(48*time.Hour), f.customer)
	f.statusCoupon(t, second, "expired", inside, inside.Add(48*time.Hour), f.customer)
	f.statusCoupon(t, second, "expired", inside, inside.Add(48*time.Hour), f.customer)

	other := newConversionFixtureIn(t, f, "other")
	var otherBenefit int64
	if err := f.pool.QueryRow(t.Context(), `SELECT id FROM benefits WHERE business_id = $1`, other.business).Scan(&otherBenefit); err != nil {
		t.Fatal(err)
	}
	other.statusCoupon(t, otherBenefit, "redeemed", inside, inside.Add(48*time.Hour), f.customer)

	got, err := postgres.NewCoupons(f.pool).BenefitPerformance(t.Context(), f.business, periodEnding(now, 7), periodEnding(time.Date(2028, 3, 8, 0, 0, 0, 0, argentina), 7), now)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Current) != 2 || got.Current[0] != (benefits.BenefitCounts{BenefitID: first, Generated: 1, Redeemed: 1}) ||
		got.Current[1] != (benefits.BenefitCounts{BenefitID: second, Generated: 2, Expired: 2}) {
		t.Errorf("current = %+v, want the two benefits of this business only", got.Current)
	}
}
