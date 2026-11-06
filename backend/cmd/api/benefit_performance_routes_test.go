package main

import (
	"net/http"
	"testing"
)

func TestBenefitPerformanceIsAProMetricWithTheRealCounts(t *testing.T) {
	t.Parallel()

	w := newMetricsWorld(t)
	starter := w.shop("starter@example.com", "starter", planStarter)
	pro := w.shop("pro@example.com", "pro", planPro)
	pilot := w.shop("pilot@example.com", "pilot", planPilot)
	other := w.shop("other@example.com", "other", planPro)

	w.coupons(starter, 30, true, 2*day)
	w.coupons(pro, 12, true, 2*day)
	w.coupons(pro, 8, false, 3*day)
	w.coupons(pro, 20, true, 40*day) // previous period: 20 of 20 redeemed, enough to compare with
	w.coupons(pilot, 25, true, 2*day)
	w.coupons(other, 90, false, 2*day)

	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/benefits", starter.token, nil); status != http.StatusForbidden || errorCode(body) != "plan_upgrade_required" || len(body) != 1 {
		t.Errorf("starter = %d %v, want only a 403 plan_upgrade_required", status, body)
	}

	status, body := w.h.do(http.MethodGet, "/me/business/metrics/benefits", pro.token, nil)
	if status != http.StatusOK || body["status"] != "ok" || body["coupons_generated"] != float64(20) {
		t.Fatalf("pro = %d %v, want 200 ok with 20 coupons", status, body)
	}
	list, _ := body["benefits"].([]any)
	if len(list) != 1 {
		t.Fatalf("benefits = %v, want the one benefit that generated coupons", list)
	}
	b := list[0].(map[string]any)
	if b["name"] != "-10%" || b["generated"] != float64(20) || b["redeemed"] != float64(12) || b["expired"] != float64(8) ||
		b["pending"] != float64(0) || b["rate"] != 0.6 || b["previous_rate"] != float64(1) || b["change"] != -0.4 {
		t.Errorf("benefit = %v, want -10%% with 20 generated, 12 redeemed, 8 expired, rate 0.6 against 1.0", b)
	}

	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/benefits", pilot.token, nil); status != http.StatusOK || body["status"] != "ok" {
		t.Errorf("pilot = %d %v, want 200 ok", status, body)
	}
	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/benefits?period=14", pro.token, nil); status != http.StatusUnprocessableEntity || errorCode(body) != "validation_failed" {
		t.Errorf("period=14 = %d %q, want 422 validation_failed", status, errorCode(body))
	}
	if status, _ := w.h.do(http.MethodGet, "/me/business/metrics/benefits", "", nil); status != http.StatusUnauthorized {
		t.Errorf("anonymous = %d, want 401", status)
	}
	assertNoPersonalData(t, "benefit performance", body)
}

func TestBenefitPerformanceWithTooFewCouponsShowsNoRate(t *testing.T) {
	t.Parallel()

	w := newMetricsWorld(t)
	pro := w.shop("pro@example.com", "pro", planPro)
	w.coupons(pro, 19, true, 2*day)

	status, body := w.h.do(http.MethodGet, "/me/business/metrics/benefits", pro.token, nil)
	if status != http.StatusOK || body["status"] != "insufficient_data" || body["coupons_generated"] != float64(19) {
		t.Fatalf("19 coupons = %d %v, want insufficient_data", status, body)
	}
	if _, present := body["benefits"]; present {
		t.Errorf("benefits are present with too few coupons: %v", body)
	}
}
