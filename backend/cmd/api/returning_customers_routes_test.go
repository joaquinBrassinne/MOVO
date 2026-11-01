package main

import (
	"net/http"
	"testing"
)

func TestReturningCustomersIsAProMetricMadeOfNumbersOnly(t *testing.T) {
	t.Parallel()

	w := newMetricsWorld(t)
	starter := w.shop("starter@example.com", "starter", planStarter)
	pro := w.shop("pro@example.com", "pro", planPro)
	pilot := w.shop("pilot@example.com", "pilot", planPilot)
	other := w.shop("other@example.com", "other", planPro)

	// All the coupons belong to the one customer of the world, so Pro has one customer who came
	// back 22 times in the period.
	w.coupons(starter, 30, true, 2*day)
	w.coupons(pro, 22, true, 2*day)
	w.coupons(pro, 3, false, 3*day)
	w.coupons(pilot, 25, true, 2*day)
	w.coupons(other, 90, true, 2*day)

	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/returning-customers", starter.token, nil); status != http.StatusForbidden || errorCode(body) != "plan_upgrade_required" || len(body) != 1 {
		t.Errorf("starter = %d %v, want only a 403 plan_upgrade_required", status, body)
	}

	status, body := w.h.do(http.MethodGet, "/me/business/metrics/returning-customers", pro.token, nil)
	if status != http.StatusOK || body["status"] != "ok" || body["customers"] != float64(1) ||
		body["one_redemption"] != float64(0) || body["two_or_more"] != float64(1) {
		t.Fatalf("pro = %d %v, want 1 customer with two or more redemptions", status, body)
	}
	if rate, _ := body["repeat_rate"].(map[string]any); rate["rate"] != float64(1) || rate["numerator"] != float64(1) || rate["denominator"] != float64(1) {
		t.Errorf("repeat_rate = %v, want 1/1", body["repeat_rate"])
	}
	assertNoPersonalData(t, "returning customers", body)

	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/returning-customers", pilot.token, nil); status != http.StatusOK || body["status"] != "ok" {
		t.Errorf("pilot = %d %v, want 200 ok", status, body)
	}
	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/returning-customers?period=14", pro.token, nil); status != http.StatusUnprocessableEntity || errorCode(body) != "validation_failed" {
		t.Errorf("period=14 = %d %q, want 422 validation_failed", status, errorCode(body))
	}
}

func TestReturningCustomersWithTooFewCouponsShowsNoCount(t *testing.T) {
	t.Parallel()

	w := newMetricsWorld(t)
	pro := w.shop("pro@example.com", "pro", planPro)
	w.coupons(pro, 19, true, 2*day)

	status, body := w.h.do(http.MethodGet, "/me/business/metrics/returning-customers", pro.token, nil)
	if status != http.StatusOK || body["status"] != "insufficient_data" || body["coupons_generated"] != float64(19) {
		t.Fatalf("19 coupons = %d %v, want insufficient_data", status, body)
	}
	for _, key := range []string{"customers", "one_redemption", "two_or_more", "repeat_rate"} {
		if _, present := body[key]; present {
			t.Errorf("%s is present with too few coupons: %v", key, body)
		}
	}
}
