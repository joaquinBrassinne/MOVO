package main

import (
	"net/http"
	"testing"
)

// reviewRedeemed gives a review of the rating to up to n redeemed coupons of the business that
// were redeemed in the window [from, to) of days ago, counted back from now.
func (w *metricsWorld) reviewRedeemed(s *metricsShop, n, rating, fromDaysAgo, toDaysAgo int) {
	w.t.Helper()
	_, err := w.h.pool.Exec(w.t.Context(), `
INSERT INTO reviews (branch_id, business_id, benefit_code_id, customer_id, rating, display_name, created_at)
SELECT redeemed_branch_id, business_id, id, customer_id, $3, 'Ana', redeemed_at + interval '1 hour'
FROM benefit_codes
WHERE business_id = $1 AND status = 'redeemed'
  AND redeemed_at <= now() - $4 * interval '1 day' AND redeemed_at > now() - $5 * interval '1 day'
  AND NOT EXISTS (SELECT 1 FROM reviews r WHERE r.benefit_code_id = benefit_codes.id)
ORDER BY id LIMIT $2`, s.business, n, rating, fromDaysAgo, toDaysAgo)
	w.must(err)
}

func TestReviewsOverTimeIsAProMetricMadeOfNumbersOnly(t *testing.T) {
	t.Parallel()

	w := newMetricsWorld(t)
	starter := w.shop("starter@example.com", "starter", planStarter)
	pro := w.shop("pro@example.com", "pro", planPro)
	pilot := w.shop("pilot@example.com", "pilot", planPilot)
	other := w.shop("other@example.com", "other", planPro)

	w.coupons(starter, 30, true, 2*day)
	w.coupons(pro, 22, true, 2*day)
	w.coupons(pro, 3, false, 3*day)
	w.coupons(pro, 20, true, 40*day) // previous period
	w.reviewRedeemed(pro, 8, 5, 1, 10)
	w.reviewRedeemed(pro, 4, 3, 1, 10)
	w.reviewRedeemed(pro, 5, 2, 35, 50) // previous period: 5 reviews of 2 stars
	w.coupons(pilot, 25, true, 2*day)
	w.coupons(other, 60, true, 2*day)
	w.reviewRedeemed(other, 60, 1, 1, 10) // must not leak into the others

	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/reviews", starter.token, nil); status != http.StatusForbidden || errorCode(body) != "plan_upgrade_required" || len(body) != 1 {
		t.Errorf("starter = %d %v, want only a 403 plan_upgrade_required", status, body)
	}

	status, body := w.h.do(http.MethodGet, "/me/business/metrics/reviews", pro.token, nil)
	if status != http.StatusOK || body["status"] != "ok" {
		t.Fatalf("pro = %d %v, want 200 ok", status, body)
	}
	rating, _ := body["rating"].(map[string]any)
	if rating["reviews"] != float64(12) || rating["previous_reviews"] != float64(5) ||
		rating["average"] != 4.33 || rating["previous_average"] != float64(2) || rating["change"] != 2.33 {
		t.Errorf("rating = %v, want 12 reviews averaging 4.33 against 5 averaging 2.0", rating)
	}
	review, _ := body["review_rate"].(map[string]any)
	if review["numerator"] != float64(12) || review["denominator"] != float64(22) ||
		review["rate"] != 0.5455 || review["previous_rate"] != 0.25 || review["change"] != 0.2955 {
		t.Errorf("review_rate = %v, want 12 of 22 = 0.5455 against 0.25", review)
	}
	assertNoPersonalData(t, "reviews over time", body)

	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/reviews", pilot.token, nil); status != http.StatusOK || body["status"] != "ok" {
		t.Errorf("pilot = %d %v, want 200 ok", status, body)
	}
	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/reviews?period=14", pro.token, nil); status != http.StatusUnprocessableEntity || errorCode(body) != "validation_failed" {
		t.Errorf("period=14 = %d %q, want 422 validation_failed", status, errorCode(body))
	}
}

func TestReviewsOverTimeWithTooFewCouponsShowsNoAverage(t *testing.T) {
	t.Parallel()

	w := newMetricsWorld(t)
	pro := w.shop("pro@example.com", "pro", planPro)
	w.coupons(pro, 19, true, 2*day)
	w.reviewRedeemed(pro, 19, 5, 1, 10)

	status, body := w.h.do(http.MethodGet, "/me/business/metrics/reviews", pro.token, nil)
	if status != http.StatusOK || body["status"] != "insufficient_data" || body["coupons_generated"] != float64(19) {
		t.Fatalf("19 coupons = %d %v, want insufficient_data", status, body)
	}
	for _, key := range []string{"rating", "review_rate"} {
		if _, present := body[key]; present {
			t.Errorf("%s is present with too few coupons: %v", key, body)
		}
	}
}
