package main

import (
	"net/http"
	"testing"
)

func TestTimeToRedeemIsAProMetricAndTheBucketsAddUp(t *testing.T) {
	t.Parallel()

	w := newMetricsWorld(t)
	starter := w.shop("starter@example.com", "starter", planStarter)
	pro := w.shop("pro@example.com", "pro", planPro)
	pilot := w.shop("pilot@example.com", "pilot", planPilot)
	other := w.shop("other@example.com", "other", planPro)

	w.coupons(starter, 30, true, 2*day)
	w.coupons(pro, 22, true, 2*day)
	w.coupons(pro, 3, false, 3*day)
	w.coupons(pilot, 25, true, 2*day)
	w.coupons(other, 90, true, 2*day)

	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/time-to-redeem", starter.token, nil); status != http.StatusForbidden || errorCode(body) != "plan_upgrade_required" || len(body) != 1 {
		t.Errorf("starter = %d %v, want only a 403 plan_upgrade_required", status, body)
	}

	status, body := w.h.do(http.MethodGet, "/me/business/metrics/time-to-redeem", pro.token, nil)
	if status != http.StatusOK || body["status"] != "ok" || body["coupons_generated"] != float64(25) || body["redemptions"] != float64(22) {
		t.Fatalf("pro = %d %v, want 200 ok with 25 coupons and 22 redemptions", status, body)
	}
	var numerators float64
	for _, key := range []string{"same_day", "within_24h", "within_48h"} {
		bucket, _ := body[key].(map[string]any)
		if bucket["denominator"] != float64(22) {
			t.Errorf("%s = %v, want 22 as the whole", key, bucket)
		}
		numerators += bucket["numerator"].(float64)
	}
	if numerators != 22 {
		t.Errorf("the buckets hold %v redemptions, want all 22", numerators)
	}

	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/time-to-redeem", pilot.token, nil); status != http.StatusOK || body["status"] != "ok" {
		t.Errorf("pilot = %d %v, want 200 ok", status, body)
	}
	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/time-to-redeem?period=14", pro.token, nil); status != http.StatusUnprocessableEntity || errorCode(body) != "validation_failed" {
		t.Errorf("period=14 = %d %q, want 422 validation_failed", status, errorCode(body))
	}
	assertNoPersonalData(t, "time to redeem", body)
}

func TestTimeToRedeemWithTooFewCouponsShowsNoShare(t *testing.T) {
	t.Parallel()

	w := newMetricsWorld(t)
	pro := w.shop("pro@example.com", "pro", planPro)
	w.coupons(pro, 19, true, 2*day)

	status, body := w.h.do(http.MethodGet, "/me/business/metrics/time-to-redeem", pro.token, nil)
	if status != http.StatusOK || body["status"] != "insufficient_data" || body["coupons_generated"] != float64(19) {
		t.Fatalf("19 coupons = %d %v, want insufficient_data", status, body)
	}
	for _, key := range []string{"same_day", "within_24h", "within_48h", "redemptions"} {
		if _, present := body[key]; present {
			t.Errorf("%s is present with too few coupons: %v", key, body)
		}
	}
}
