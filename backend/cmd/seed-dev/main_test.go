package main

import (
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

// TestSeedGivesTheDemoBusinessBenefitsCouponsAndCounters runs both scripts twice against a
// migrated database: the demo owner must find something in every view of the panel, and
// running again must not duplicate anything.
func TestSeedGivesTheDemoBusinessBenefitsCouponsAndCounters(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()

	for range 2 {
		for _, script := range []string{seedSQL, seedCouponsSQL} {
			if _, err := pool.Exec(ctx, script); err != nil {
				t.Fatalf("seed: %v", err)
			}
		}
	}

	var (
		limit                                               *int
		benefits, pending, redeemed, expired, views, clicks int
	)
	err := pool.QueryRow(ctx, `
SELECT b.monthly_coupon_limit,
  (SELECT count(*) FROM benefits WHERE business_id = b.id),
  (SELECT count(*) FROM benefit_codes WHERE business_id = b.id AND status = 'pending'),
  (SELECT count(*) FROM benefit_codes WHERE business_id = b.id AND status = 'redeemed'),
  (SELECT count(*) FROM benefit_codes WHERE business_id = b.id AND status = 'expired'),
  (SELECT count(*) FROM metric_daily WHERE business_id = b.id AND metric = 'profile_view'),
  (SELECT count(*) FROM metric_daily WHERE business_id = b.id AND metric = 'whatsapp_click')
FROM businesses b WHERE b.slug = 'alma-haus-training-club'`).Scan(&limit, &benefits, &pending, &redeemed, &expired, &views, &clicks)
	if err != nil {
		t.Fatalf("read the demo business: %v", err)
	}

	if limit == nil || *limit != 30 || benefits != 3 || pending != 3 || redeemed != 3 || expired != 1 || views != 7 || clicks != 7 {
		t.Errorf("limit %v, %d benefits, %d pending, %d redeemed, %d expired, %d view days, %d click days; want 30, 3, 3, 3, 1, 7, 7",
			limit, benefits, pending, redeemed, expired, views, clicks)
	}
}
