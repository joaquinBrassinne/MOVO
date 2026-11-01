package main

import (
	"net/http"
	"testing"
)

func TestBranchComparisonIsAProMetricWithTheRealCounts(t *testing.T) {
	t.Parallel()

	w := newMetricsWorld(t)
	starter := w.shop("starter@example.com", "starter", planStarter)
	pro := w.shop("pro@example.com", "pro", planPro)
	pilot := w.shop("pilot@example.com", "pilot", planPilot)
	other := w.shop("other@example.com", "other", planPro)
	var yoga int64
	w.must(w.h.pool.QueryRow(t.Context(), `INSERT INTO services (business_id, name) VALUES ($1, 'Yoga') RETURNING id`, pro.business).Scan(&yoga))

	w.coupons(starter, 30, true, 2*day)
	w.coupons(pro, 22, true, 2*day)
	w.coupons(pro, 3, false, 3*day)
	w.coupons(pro, 20, true, 40*day) // previous period, enough to compare with
	w.counter(pro, "service_view", 3, pro.branch, yoga, 30)
	w.counter(pro, "service_view", 45, pro.branch, yoga, 20)
	w.counter(pro, "whatsapp_click", 3, pro.branch, 0, 9)
	w.coupons(pilot, 25, true, 2*day)
	w.coupons(other, 90, true, 2*day)
	w.counter(other, "whatsapp_click", 3, other.branch, 0, 999)

	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/branches", starter.token, nil); status != http.StatusForbidden || errorCode(body) != "plan_upgrade_required" || len(body) != 1 {
		t.Errorf("starter = %d %v, want only a 403 plan_upgrade_required", status, body)
	}

	status, body := w.h.do(http.MethodGet, "/me/business/metrics/branches", pro.token, nil)
	if status != http.StatusOK || body["status"] != "ok" {
		t.Fatalf("pro = %d %v, want 200 ok", status, body)
	}
	branches, _ := body["branches"].([]any)
	if len(branches) != 1 {
		t.Fatalf("branches = %v, want the one branch of this business", branches)
	}
	b := branches[0].(map[string]any)
	views, taps, redeemed := b["service_views"].(map[string]any), b["whatsapp_clicks"].(map[string]any), b["redemptions"].(map[string]any)
	if b["name"] != "Centro" || b["active"] != true ||
		views["value"] != float64(30) || views["previous"] != float64(20) || views["change"] != 0.5 ||
		taps["value"] != float64(9) || taps["previous"] != float64(0) || taps["change"] != nil ||
		redeemed["value"] != float64(22) || redeemed["previous"] != float64(20) || redeemed["change"] != 0.1 {
		t.Errorf("branch = %v, want Centro: 30 views (+50%%), 9 taps (no baseline), 22 redemptions (+10%%)", b)
	}

	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/branches", pilot.token, nil); status != http.StatusOK || body["status"] != "ok" {
		t.Errorf("pilot = %d %v, want 200 ok", status, body)
	}
	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/branches?period=14", pro.token, nil); status != http.StatusUnprocessableEntity || errorCode(body) != "validation_failed" {
		t.Errorf("period=14 = %d %q, want 422 validation_failed", status, errorCode(body))
	}
	assertNoPersonalData(t, "branch comparison", body)
}

func TestBranchComparisonWithTooFewCouponsShowsNoBranches(t *testing.T) {
	t.Parallel()

	w := newMetricsWorld(t)
	pro := w.shop("pro@example.com", "pro", planPro)
	w.coupons(pro, 19, true, 2*day)

	status, body := w.h.do(http.MethodGet, "/me/business/metrics/branches", pro.token, nil)
	if status != http.StatusOK || body["status"] != "insufficient_data" || body["coupons_generated"] != float64(19) {
		t.Fatalf("19 coupons = %d %v, want insufficient_data", status, body)
	}
	if _, present := body["branches"]; present {
		t.Errorf("branches are present with too few coupons: %v", body)
	}
}
