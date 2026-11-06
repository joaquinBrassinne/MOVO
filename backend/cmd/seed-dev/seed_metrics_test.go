package main

import (
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

// TestSeedMetricsGivesTheDemoBusinessAProHistory runs every script twice: the demo business must
// end on Pro with a year of redemptions, a ticket, an alert and a super admin, and running again
// must not duplicate anything.
func TestSeedMetricsGivesTheDemoBusinessAProHistory(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()

	for range 2 {
		for _, script := range []string{seedSQL, seedCouponsSQL, seedReviewsSQL, seedMetricsSQL} {
			if _, err := pool.Exec(ctx, script); err != nil {
				t.Fatalf("seed: %v", err)
			}
		}
	}

	var (
		plan                                                string
		ticket                                              *int
		history, monthsWithRedemptions, alerts, serviceDays int
		admins                                              int
	)
	err := pool.QueryRow(ctx, `
SELECT s.plan, b.average_ticket_ars,
  (SELECT count(*) FROM benefit_codes WHERE business_id = b.id AND code LIKE 'MT%'),
  (SELECT count(DISTINCT date_trunc('month', redeemed_at)) FROM benefit_codes WHERE business_id = b.id AND status = 'redeemed'),
  (SELECT count(*) FROM alerts WHERE business_id = b.id AND type = 'unredeemed_coupons'),
  (SELECT count(*) FROM metric_daily WHERE business_id = b.id AND metric = 'service_view'),
  (SELECT count(*) FROM users WHERE role = 'super_admin')
FROM businesses b JOIN subscriptions s ON s.business_id = b.id
WHERE b.slug = 'alma-haus-training-club'`).Scan(&plan, &ticket, &history, &monthsWithRedemptions, &alerts, &serviceDays, &admins)
	if err != nil {
		t.Fatalf("read the demo business: %v", err)
	}
	if plan != "pro" || ticket == nil || *ticket != 6500 || history < 900 || monthsWithRedemptions < 13 || alerts != 1 || serviceDays == 0 || admins != 1 {
		t.Errorf("plan %q, ticket %v, %d historic coupons, %d months with redemptions, %d alerts, %d service view rows, %d admins; want pro, 6500, 900+, 13+, 1, some, 1",
			plan, ticket, history, monthsWithRedemptions, alerts, serviceDays, admins)
	}
}
