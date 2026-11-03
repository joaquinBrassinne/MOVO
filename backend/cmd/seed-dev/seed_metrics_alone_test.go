package main

import (
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

// TestSeedMetricsWorksWhenTheCouponSeedWasSkipped covers a database where the demo owner already
// made a benefit of their own, so seed_coupons.sql does nothing and no demo customer exists.
func TestSeedMetricsWorksWhenTheCouponSeedWasSkipped(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()

	if _, err := pool.Exec(ctx, seedSQL); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO benefits (business_id, type, percent)
		SELECT id, 'percent', 10 FROM businesses WHERE slug = 'alma-haus-training-club'`); err != nil {
		t.Fatal(err)
	}
	// The owner's own counters of earlier days must not make the seed fail.
	if _, err := pool.Exec(ctx, `INSERT INTO metric_daily (business_id, branch_id, date, metric, count)
		SELECT id, NULL, current_date - 20, 'profile_view', 5 FROM businesses WHERE slug = 'alma-haus-training-club'`); err != nil {
		t.Fatal(err)
	}
	for _, script := range []string{seedCouponsSQL, seedMetricsSQL} {
		if _, err := pool.Exec(ctx, script); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	var history int
	var ticket *int
	if err := pool.QueryRow(ctx, `
SELECT (SELECT count(*) FROM benefit_codes WHERE business_id = b.id AND code LIKE 'MT%'), b.average_ticket_ars
FROM businesses b WHERE b.slug = 'alma-haus-training-club'`).Scan(&history, &ticket); err != nil {
		t.Fatal(err)
	}
	if history < 900 || ticket == nil || *ticket != 6500 {
		t.Errorf("%d historic coupons and ticket %v; want 900+ and 6500 even without the coupon seed", history, ticket)
	}
}
