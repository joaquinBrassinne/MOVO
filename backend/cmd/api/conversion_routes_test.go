package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestConversionIsAProMetricAndStarterGetsNoData(t *testing.T) {
	t.Parallel()

	w := newMetricsWorld(t)
	starter := w.shop("starter@example.com", "starter", planStarter)
	pro := w.shop("pro@example.com", "pro", planPro)
	pilot := w.shop("pilot@example.com", "pilot", planPilot)
	stranger := w.h.user("stranger@example.com", "business_admin")
	w.coupons(starter, 30, true, 2*day)
	w.coupons(pro, 30, true, 2*day)

	// A Starter gets the stable code and a message, and none of the numbers.
	status, body := w.h.do(http.MethodGet, "/me/business/metrics/conversion", starter.token, nil)
	if status != http.StatusForbidden || errorCode(body) != "plan_upgrade_required" {
		t.Fatalf("starter = %d %v, want 403 plan_upgrade_required", status, body)
	}
	if len(body) != 1 {
		t.Errorf("starter body = %v, want only the error", body)
	}
	if msg, _ := body["error"].(map[string]any)["message"].(string); !strings.Contains(strings.ToLower(msg), "upgrade") {
		t.Errorf("message = %q, want it to invite to upgrade", msg)
	}

	// Pro and a pilot (which has the Pro plan) are served.
	for name, s := range map[string]*metricsShop{"pro": pro, "pilot": pilot} {
		if status, body := w.h.do(http.MethodGet, "/me/business/metrics/conversion", s.token, nil); status != http.StatusOK {
			t.Errorf("%s = %d %v, want 200", name, status, body)
		}
	}

	// A downgrade closes the door again, with the same answer.
	w.setPlan(pro, planStarter)
	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/conversion", pro.token, nil); status != http.StatusForbidden || errorCode(body) != "plan_upgrade_required" {
		t.Errorf("downgraded = %d %v, want 403 plan_upgrade_required", status, body)
	}

	if status, _ := w.h.do(http.MethodGet, "/me/business/metrics/conversion", "", nil); status != http.StatusUnauthorized {
		t.Errorf("anonymous = %d, want 401", status)
	}
	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/conversion", w.admin, nil); status != http.StatusForbidden || errorCode(body) != "forbidden" {
		t.Errorf("super admin = %d %q, want 403 forbidden", status, errorCode(body))
	}
	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/conversion", stranger, nil); status != http.StatusNotFound || errorCode(body) != "business_not_found" {
		t.Errorf("owner without a business = %d %q, want 404 business_not_found", status, errorCode(body))
	}
}

func TestConversionWithTooFewCouponsShowsNoPercentage(t *testing.T) {
	t.Parallel()

	w := newMetricsWorld(t)
	pro := w.shop("pro@example.com", "pro", planPro)
	w.coupons(pro, 19, true, 2*day)
	w.counter(pro, "profile_view", 2, 0, 0, 100)

	status, body := w.h.do(http.MethodGet, "/me/business/metrics/conversion", pro.token, nil)
	if status != http.StatusOK || body["status"] != "insufficient_data" || body["coupons_generated"] != float64(19) || body["minimum_sample"] != float64(20) {
		t.Fatalf("19 coupons = %d %v, want insufficient_data", status, body)
	}
	for _, key := range []string{"views_to_coupons", "coupons_to_redemptions", "by_branch", "by_activity"} {
		if _, present := body[key]; present {
			t.Errorf("%s is present with too few coupons: %v", key, body)
		}
	}

	w.coupons(pro, 1, false, 3*day)
	if _, body := w.h.do(http.MethodGet, "/me/business/metrics/conversion", pro.token, nil); body["status"] != "ok" {
		t.Errorf("20 coupons = %v, want ok", body)
	}
}

func TestConversionRatesTrendAndBreakdownMatchTheData(t *testing.T) {
	t.Parallel()

	w := newMetricsWorld(t)
	pro := w.shop("pro@example.com", "pro", planPro)
	other := w.shop("other@example.com", "other", planPro)
	ctx := t.Context()
	var yoga int64
	w.must(w.h.pool.QueryRow(ctx, `INSERT INTO services (business_id, name) VALUES ($1, 'Yoga') RETURNING id`, pro.business).Scan(&yoga))

	// Last 30 days: 30 coupons, 10 redeemed, 120 profile views. The 30 before: 25 coupons,
	// 5 redeemed, 100 views.
	w.coupons(pro, 20, false, 5*day)
	w.coupons(pro, 10, true, 6*day)
	w.coupons(pro, 20, false, 40*day)
	w.coupons(pro, 5, true, 41*day)
	w.counter(pro, "profile_view", 3, 0, 0, 120)
	w.counter(pro, "profile_view", 45, 0, 0, 100)
	w.counter(pro, "whatsapp_click", 3, pro.branch, 0, 8)
	w.counter(pro, "service_view", 3, pro.branch, yoga, 20)
	w.counter(pro, "whatsapp_click", 3, pro.branch, yoga, 5)

	// Another business with huge numbers must change nothing here.
	w.coupons(other, 50, true, 2*day)
	w.counter(other, "profile_view", 2, 0, 0, 9999)

	status, body := w.h.do(http.MethodGet, "/me/business/metrics/conversion", pro.token, nil)
	if status != http.StatusOK || body["status"] != "ok" {
		t.Fatalf("conversion = %d %v, want 200 ok", status, body)
	}
	rate := func(key string) map[string]any {
		m, _ := body[key].(map[string]any)
		return m
	}
	vtc, ctr := rate("views_to_coupons"), rate("coupons_to_redemptions")
	if vtc["numerator"] != float64(30) || vtc["denominator"] != float64(120) || vtc["rate"] != 0.25 || vtc["previous_rate"] != 0.25 || vtc["change"] != float64(0) {
		t.Errorf("views to coupons = %v, want 30/120 = 0.25 against 0.25", vtc)
	}
	if ctr["numerator"] != float64(10) || ctr["denominator"] != float64(30) || ctr["rate"] != 0.3333 || ctr["previous_rate"] != 0.2 || ctr["change"] != 0.1333 {
		t.Errorf("coupons to redemptions = %v, want 10/30 = 0.3333 against 0.2, change 0.1333", ctr)
	}

	branches, _ := body["by_branch"].([]any)
	if len(branches) != 1 {
		t.Fatalf("by_branch = %v, want the one branch", branches)
	}
	if b := branches[0].(map[string]any); b["name"] != "Centro" || b["whatsapp_clicks"] != float64(13) || b["redemptions"] != float64(10) || b["rate"] != 0.7692 {
		t.Errorf("branch = %v, want Centro with 13 taps and 10 redemptions", b)
	}
	activities, _ := body["by_activity"].([]any)
	if len(activities) != 1 {
		t.Fatalf("by_activity = %v, want Yoga", activities)
	}
	if a := activities[0].(map[string]any); a["name"] != "Yoga" || a["views"] != float64(20) || a["whatsapp_clicks"] != float64(5) || a["rate"] != 0.25 {
		t.Errorf("activity = %v, want Yoga with 20 views and 5 taps", a)
	}

	// The shorter periods are cut from the same data.
	_, week := w.h.do(http.MethodGet, "/me/business/metrics/conversion?period=7", pro.token, nil)
	if week["status"] != "ok" || week["coupons_generated"] != float64(30) {
		t.Errorf("7 days = %v, want the 30 coupons of the last week", week)
	}
	if period, _ := week["period"].(map[string]any); period["days"] != float64(7) {
		t.Errorf("period = %v, want 7 days", period)
	}
	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/conversion?period=14", pro.token, nil); status != http.StatusUnprocessableEntity || errorCode(body) != "validation_failed" {
		t.Errorf("period=14 = %d %q, want 422 validation_failed", status, errorCode(body))
	}
}

func TestNoProMetricShowsAnyNameOrPhoneOfACustomer(t *testing.T) {
	t.Parallel()

	w := newMetricsWorld(t)
	pro := w.shop("pro@example.com", "pro", planPro)
	w.coupons(pro, 30, true, 2*day)
	w.counter(pro, "profile_view", 2, 0, 0, 50)

	for _, path := range []string{"/me/business/metrics/conversion"} {
		status, body := w.h.do(http.MethodGet, path, pro.token, nil)
		if status != http.StatusOK {
			t.Fatalf("%s = %d %v", path, status, body)
		}
		assertNoPersonalData(t, path, body)
	}
}
