package main

import (
	"net/http"
	"testing"
)

func (w *metricsWorld) setTicket(s *metricsShop, ticket any) {
	w.t.Helper()
	_, err := w.h.pool.Exec(w.t.Context(), `UPDATE businesses SET average_ticket_ars = $2 WHERE id = $1`, s.business, ticket)
	w.must(err)
}

func TestEstimatedReturnNeedsTheProPlanAndATicketAndAlwaysSaysEstimated(t *testing.T) {
	t.Parallel()

	w := newMetricsWorld(t)
	starter := w.shop("starter@example.com", "starter", planStarter)
	pro := w.shop("pro@example.com", "pro", planPro)
	pilot := w.shop("pilot@example.com", "pilot", planPilot)
	w.coupons(starter, 30, true, 2*day)
	w.coupons(pro, 22, true, 2*day)
	w.coupons(pro, 3, false, 3*day)
	w.coupons(pro, 20, true, 40*day) // previous period: 20 redemptions
	w.coupons(pilot, 25, true, 2*day)
	w.setTicket(starter, 15000) // a downgraded business keeps its ticket but may not read the metric
	w.setTicket(pilot, 10000)

	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/estimated-return", starter.token, nil); status != http.StatusForbidden || errorCode(body) != "plan_upgrade_required" || len(body) != 1 {
		t.Errorf("starter = %d %v, want only a 403 plan_upgrade_required", status, body)
	}

	// Pro without a ticket: not shown, whatever the coupons.
	status, body := w.h.do(http.MethodGet, "/me/business/metrics/estimated-return", pro.token, nil)
	if status != http.StatusOK || body["status"] != "no_ticket" {
		t.Fatalf("no ticket = %d %v, want no_ticket", status, body)
	}
	for _, key := range []string{"estimated_return_ars", "plan_price_ars", "average_ticket_ars", "redemptions"} {
		if _, present := body[key]; present {
			t.Errorf("%s is shown without a ticket: %v", key, body)
		}
	}

	w.setTicket(pro, 15000)
	status, body = w.h.do(http.MethodGet, "/me/business/metrics/estimated-return", pro.token, nil)
	if status != http.StatusOK || body["status"] != "ok" || body["estimated"] != true ||
		body["redemptions"] != float64(22) || body["average_ticket_ars"] != float64(15000) ||
		body["estimated_return_ars"] != float64(330000) || body["plan_price_ars"] != float64(29800) ||
		body["previous_estimated_return_ars"] != float64(300000) {
		t.Fatalf("pro = %d %v, want an estimate of 22 x 15000 = 330000 beside 29800, against 300000", status, body)
	}
	assertNoPersonalData(t, "estimated return", body)

	// A pilot pays nothing but is on the Pro plan: it reads the metric.
	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/estimated-return", pilot.token, nil); status != http.StatusOK || body["estimated_return_ars"] != float64(250000) {
		t.Errorf("pilot = %d %v, want 25 x 10000 = 250000", status, body)
	}
	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/estimated-return?period=14", pro.token, nil); status != http.StatusUnprocessableEntity || errorCode(body) != "validation_failed" {
		t.Errorf("period=14 = %d %q, want 422 validation_failed", status, errorCode(body))
	}
}

func TestEstimatedReturnWithTooFewCouponsShowsNoFigure(t *testing.T) {
	t.Parallel()

	w := newMetricsWorld(t)
	pro := w.shop("pro@example.com", "pro", planPro)
	w.setTicket(pro, 15000)
	w.coupons(pro, 19, true, 2*day)

	status, body := w.h.do(http.MethodGet, "/me/business/metrics/estimated-return", pro.token, nil)
	if status != http.StatusOK || body["status"] != "insufficient_data" || body["coupons_generated"] != float64(19) {
		t.Fatalf("19 coupons = %d %v, want insufficient_data", status, body)
	}
	if _, present := body["estimated_return_ars"]; present {
		t.Errorf("a figure is shown with too few coupons: %v", body)
	}
}
