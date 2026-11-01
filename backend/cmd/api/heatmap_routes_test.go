package main

import (
	"net/http"
	"testing"
)

func TestHeatmapIsAProMetricAndAddsUp(t *testing.T) {
	t.Parallel()

	w := newMetricsWorld(t)
	starter := w.shop("starter@example.com", "starter", planStarter)
	pro := w.shop("pro@example.com", "pro", planPro)
	pilot := w.shop("pilot@example.com", "pilot", planPilot)
	other := w.shop("other@example.com", "other", planPro)
	w.coupons(starter, 30, true, 2*day)
	w.coupons(pro, 25, true, 2*day)
	w.coupons(pro, 5, false, 3*day)
	w.coupons(pro, 4, true, 40*day) // previous period
	w.coupons(pilot, 25, true, 2*day)
	w.coupons(other, 60, true, 2*day) // must not leak into the others

	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/heatmap", starter.token, nil); status != http.StatusForbidden || errorCode(body) != "plan_upgrade_required" || len(body) != 1 {
		t.Errorf("starter = %d %v, want only a 403 plan_upgrade_required", status, body)
	}

	status, body := w.h.do(http.MethodGet, "/me/business/metrics/heatmap", pro.token, nil)
	if status != http.StatusOK || body["status"] != "ok" {
		t.Fatalf("pro = %d %v, want 200 ok", status, body)
	}
	cells, _ := body["cells"].([]any)
	if len(cells) != 28 {
		t.Fatalf("cells = %d, want a full week of four slots", len(cells))
	}
	var sum float64
	nonZero := 0
	for i, raw := range cells {
		c := raw.(map[string]any)
		if c["weekday"] != float64(i/4+1) || c["slot"] != float64(i%4) {
			t.Fatalf("cell %d = %v, want weekday %d slot %d", i, c, i/4+1, i%4)
		}
		if n := c["redemptions"].(float64); n > 0 {
			sum += n
			nonZero++
		}
	}
	// The 25 redeemed coupons of Pro share one instant, so they land in one cell.
	if sum != 25 || nonZero != 1 || body["redemptions"] != float64(25) || body["previous_redemptions"] != float64(4) {
		t.Errorf("sum %v in %d cells, redemptions %v, previous %v; want 25 in 1 cell and 4 before", sum, nonZero, body["redemptions"], body["previous_redemptions"])
	}
	if slots, _ := body["slots"].([]any); len(slots) != 4 || slots[1].(map[string]any)["start"] != "06:00" || slots[1].(map[string]any)["end"] != "11:59" {
		t.Errorf("slots = %v, want four, the second from 06:00 to 11:59", body["slots"])
	}

	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/heatmap", pilot.token, nil); status != http.StatusOK || body["status"] != "ok" {
		t.Errorf("pilot = %d %v, want 200 ok", status, body)
	}
	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/heatmap?period=14", pro.token, nil); status != http.StatusUnprocessableEntity || errorCode(body) != "validation_failed" {
		t.Errorf("period=14 = %d %q, want 422 validation_failed", status, errorCode(body))
	}
	if status, _ := w.h.do(http.MethodGet, "/me/business/metrics/heatmap", "", nil); status != http.StatusUnauthorized {
		t.Errorf("anonymous = %d, want 401", status)
	}
}

func TestHeatmapWithTooFewCouponsAndWithoutPersonalData(t *testing.T) {
	t.Parallel()

	w := newMetricsWorld(t)
	pro := w.shop("pro@example.com", "pro", planPro)
	w.coupons(pro, 19, true, 2*day)

	status, body := w.h.do(http.MethodGet, "/me/business/metrics/heatmap", pro.token, nil)
	if status != http.StatusOK || body["status"] != "insufficient_data" || body["coupons_generated"] != float64(19) {
		t.Fatalf("19 coupons = %d %v, want insufficient_data", status, body)
	}
	for _, key := range []string{"cells", "slots", "redemptions"} {
		if _, present := body[key]; present {
			t.Errorf("%s is present with too few coupons: %v", key, body)
		}
	}

	w.coupons(pro, 1, true, 2*day)
	_, body = w.h.do(http.MethodGet, "/me/business/metrics/heatmap", pro.token, nil)
	assertNoPersonalData(t, "heatmap", body)
}
