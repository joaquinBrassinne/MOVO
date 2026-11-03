package main

import (
	"net/http"
	"testing"
	"time"
)

// setQuota gives the business a monthly limit.
func (w *metricsWorld) setQuota(s *metricsShop, limit any) {
	w.t.Helper()
	_, err := w.h.pool.Exec(w.t.Context(), `UPDATE businesses SET monthly_coupon_limit = $2 WHERE id = $1`, s.business, limit)
	w.must(err)
}

// couponsThisMonth seeds n coupons generated at the start of the month on Argentina's clock, so
// they count for the month whatever day the test runs on.
func (w *metricsWorld) couponsThisMonth(s *metricsShop, n int) {
	w.t.Helper()
	const letters = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	for range n {
		w.codes++
		code := "XXXXXX" + string(letters[w.codes/len(letters)]) + string(letters[w.codes%len(letters)])
		hash := make([]byte, 32)
		copy(hash, code)
		_, err := w.h.pool.Exec(w.t.Context(), `
INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, status, expires_at, created_at)
VALUES ($1, $2, $3, (SELECT id FROM customers LIMIT 1), $4, 'expired', now() + interval '1 day',
        date_trunc('month', now() AT TIME ZONE 'America/Argentina/Cordoba') AT TIME ZONE 'America/Argentina/Cordoba' + interval '1 minute')`,
			code, s.benefit, s.business, hash)
		w.must(err)
	}
}

func TestQuotaProjectionIsAProMetricThatNeedsAQuota(t *testing.T) {
	t.Parallel()

	w := newMetricsWorld(t)
	starter := w.shop("starter@example.com", "starter", planStarter)
	pro := w.shop("pro@example.com", "pro", planPro)
	pilot := w.shop("pilot@example.com", "pilot", planPilot)
	w.setQuota(starter, 100)

	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/quota-projection", starter.token, nil); status != http.StatusForbidden || errorCode(body) != "plan_upgrade_required" || len(body) != 1 {
		t.Errorf("starter = %d %v, want only a 403 plan_upgrade_required", status, body)
	}

	// No quota: nothing to run out of, and nothing else in the answer.
	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/quota-projection", pro.token, nil); status != http.StatusOK || body["status"] != "no_quota" || len(body) != 1 {
		t.Errorf("no quota = %d %v, want only {status: no_quota}", status, body)
	}
	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/quota-projection", pilot.token, nil); status != http.StatusOK || body["status"] != "no_quota" {
		t.Errorf("pilot = %d %v, want 200 no_quota", status, body)
	}

	if status, _ := w.h.do(http.MethodGet, "/me/business/metrics/quota-projection", "", nil); status != http.StatusUnauthorized {
		t.Errorf("anonymous = %d, want 401", status)
	}
	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/quota-projection", w.admin, nil); status != http.StatusForbidden || errorCode(body) != "forbidden" {
		t.Errorf("super admin = %d %q, want 403 forbidden", status, errorCode(body))
	}
}

func TestQuotaProjectionFollowsTheCouponsOfTheMonth(t *testing.T) {
	t.Parallel()

	w := newMetricsWorld(t)
	pro := w.shop("pro@example.com", "pro", planPro)
	other := w.shop("other@example.com", "other", planPro)
	w.setQuota(pro, 100)
	w.setQuota(other, 5)
	w.couponsThisMonth(other, 5) // the quota of another business must not matter here

	// Nothing generated yet: no estimate.
	status, body := w.h.do(http.MethodGet, "/me/business/metrics/quota-projection", pro.token, nil)
	if status != http.StatusOK || body["status"] != "insufficient_data" || body["minimum_days"] != float64(3) || body["limit"] != float64(100) || body["generated"] != float64(0) {
		t.Fatalf("empty month = %d %v, want insufficient_data", status, body)
	}

	w.couponsThisMonth(pro, 100)
	status, body = w.h.do(http.MethodGet, "/me/business/metrics/quota-projection", pro.token, nil)
	if status != http.StatusOK || body["status"] != "exhausted" || body["generated"] != float64(100) || body["limit"] != float64(100) {
		t.Fatalf("used up = %d %v, want exhausted", status, body)
	}
	if _, present := body["exhausts_on"]; present {
		t.Errorf("a quota already used up has no day to run out: %v", body)
	}
	if reset, _ := body["resets_on"].(string); len(reset) != 10 || reset[8:] != "01" {
		t.Errorf("resets_on = %v, want the first day of a month", body["resets_on"])
	}
	assertNoPersonalData(t, "quota projection", body)

	// Late in the month an estimate is made from the pace so far. The test cannot choose the
	// day, so it only checks the shape: on the first two days there is no estimate, later there is.
	w.setQuota(pro, 1000)
	_, body = w.h.do(http.MethodGet, "/me/business/metrics/quota-projection", pro.token, nil)
	day := time.Now().In(time.FixedZone("ART", -3*60*60)).Day()
	switch {
	case day < 3 && body["status"] != "insufficient_data":
		t.Errorf("day %d = %v, want insufficient_data", day, body)
	case day >= 3 && body["status"] != "on_track" && body["status"] != "will_run_out":
		t.Errorf("day %d = %v, want an estimate", day, body)
	}
}
