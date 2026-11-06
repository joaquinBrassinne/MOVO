package main

import (
	"net/http"
	"testing"
	"time"
)

var argentinaClock = time.FixedZone("ART", -3*60*60)

// couponsInMonth seeds n redeemed coupons in the middle of the calendar month monthsAgo months
// before the current one, on Argentina's calendar.
func (w *metricsWorld) couponsInMonth(s *metricsShop, monthsAgo, n int) {
	w.t.Helper()
	const letters = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	now := time.Now().In(argentinaClock)
	middle := time.Date(now.Year(), now.Month(), 1, 12, 0, 0, 0, argentinaClock).AddDate(0, -monthsAgo, 14)
	for range n {
		w.codes++
		code := "WWWWWW" + string(letters[w.codes/len(letters)]) + string(letters[w.codes%len(letters)])
		hash := make([]byte, 32)
		copy(hash, code)
		_, err := w.h.pool.Exec(w.t.Context(), `
INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, status, expires_at, created_at, redeemed_at, redeemed_by, redeemed_branch_id)
VALUES ($1, $2, $3, (SELECT id FROM customers LIMIT 1), $4, 'redeemed', $5, $5, $5::timestamptz + interval '1 minute', $6, $7)`,
			code, s.benefit, s.business, hash, middle, s.owner, s.branch)
		w.must(err)
	}
}

func TestRecordsAreAProMetricRankedOverTheLastTwelveCompleteMonths(t *testing.T) {
	t.Parallel()

	w := newMetricsWorld(t)
	starter := w.shop("starter@example.com", "starter", planStarter)
	pro := w.shop("pro@example.com", "pro", planPro)
	pilot := w.shop("pilot@example.com", "pilot", planPilot)
	other := w.shop("other@example.com", "other", planPro)

	// Four months back the business starts; the two months before the last tie at 30, and the
	// most recent of them is the record. This month it has 12 so far.
	w.couponsInMonth(pro, 4, 5)
	w.couponsInMonth(pro, 3, 30)
	w.couponsInMonth(pro, 2, 30)
	w.couponsInMonth(pro, 1, 25)
	w.coupons(pro, 12, true, time.Minute) // this month
	w.couponsInMonth(other, 2, 80)        // another business must not leak in
	w.couponsInMonth(starter, 2, 40)
	w.couponsInMonth(pilot, 2, 21)
	w.couponsInMonth(pilot, 3, 3)

	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/records", starter.token, nil); status != http.StatusForbidden || errorCode(body) != "plan_upgrade_required" || len(body) != 1 {
		t.Errorf("starter = %d %v, want only a 403 plan_upgrade_required", status, body)
	}

	status, body := w.h.do(http.MethodGet, "/me/business/metrics/records", pro.token, nil)
	if status != http.StatusOK || body["status"] != "ok" {
		t.Fatalf("pro = %d %v, want 200 ok", status, body)
	}
	now := time.Now().In(argentinaClock)
	label := func(monthsAgo int) string {
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, argentinaClock).AddDate(0, -monthsAgo, 0).Format("2006-01")
	}
	best, _ := body["best_month"].(map[string]any)
	if best["month"] != label(2) || best["redemptions"] != float64(30) {
		t.Errorf("best_month = %v, want %s with 30 (the more recent of the two tied)", best, label(2))
	}
	current, _ := body["current_month"].(map[string]any)
	if current["month"] != label(0) || current["redemptions"] != float64(12) || current["progress"] != 0.4 || current["record_in_progress"] != false {
		t.Errorf("current_month = %v, want %s at 12 of 30 (0.4)", current, label(0))
	}
	if body["history_months"] != float64(3) || body["minimum_history_months"] != float64(2) || body["minimum_best_month_redemptions"] != float64(20) {
		t.Errorf("history = %v of %v, minimum best month %v; want 3 complete months of 2 needed, 20 redemptions", body["history_months"], body["minimum_history_months"], body["minimum_best_month_redemptions"])
	}
	if star, _ := body["star_benefit"].(map[string]any); star["name"] != "-10%" || star["redemptions"] != float64(90) {
		t.Errorf("star_benefit = %v, want -10%% with the 90 redemptions of the last twelve months", star)
	}
	if weekday, _ := body["strong_weekday"].(float64); weekday < 1 || weekday > 7 {
		t.Errorf("strong_weekday = %v, want an ISO weekday", body["strong_weekday"])
	}
	if slot, _ := body["strong_slot"].(map[string]any); slot["key"] != "afternoon" && slot["key"] != "morning" && slot["key"] != "evening" && slot["key"] != "overnight" {
		t.Errorf("strong_slot = %v, want one of the four slots", body["strong_slot"])
	}
	assertNoPersonalData(t, "records", body)

	// A pilot with only 21 redemptions in its best month is enough; with one month of history it is not.
	if status, body := w.h.do(http.MethodGet, "/me/business/metrics/records", pilot.token, nil); status != http.StatusOK || body["status"] != "ok" {
		t.Errorf("pilot = %d %v, want 200 ok", status, body)
	}
	if status, _ := w.h.do(http.MethodGet, "/me/business/metrics/records", "", nil); status != http.StatusUnauthorized {
		t.Errorf("anonymous = %d, want 401", status)
	}
}

func TestRecordsNeedHistoryAndASizeableBestMonth(t *testing.T) {
	t.Parallel()

	w := newMetricsWorld(t)
	young := w.shop("young@example.com", "young", planPro)
	small := w.shop("small@example.com", "small", planPro)

	// A business that started last month has a single complete month, however big.
	w.couponsInMonth(young, 1, 60)
	// A business with history whose best month has 19.
	w.couponsInMonth(small, 3, 5)
	w.couponsInMonth(small, 2, 19)
	w.couponsInMonth(small, 1, 10)

	status, body := w.h.do(http.MethodGet, "/me/business/metrics/records", young.token, nil)
	if status != http.StatusOK || body["status"] != "insufficient_data" || body["history_months"] != float64(0) {
		t.Fatalf("young = %d %v, want insufficient_data with no complete month of history", status, body)
	}
	for _, key := range []string{"best_month", "current_month", "star_benefit", "strong_weekday"} {
		if _, present := body[key]; present {
			t.Errorf("%s is present with too little history: %v", key, body)
		}
	}

	status, body = w.h.do(http.MethodGet, "/me/business/metrics/records", small.token, nil)
	if status != http.StatusOK || body["status"] != "insufficient_data" || body["best_month_redemptions"] != float64(19) || body["history_months"] != float64(2) {
		t.Fatalf("small = %d %v, want insufficient_data with a best month of 19 and 2 complete months of history", status, body)
	}
}
