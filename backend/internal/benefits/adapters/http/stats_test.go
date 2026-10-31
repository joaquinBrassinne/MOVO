package benefitshttp_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestBenefitListShowsWhatEachOneDidThisMonth(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	path := s.visibleBenefit("owner@example.com", 10)
	owner := "token-owner@example.com"
	second := s.do(http.MethodPost, "/me/business/benefits", owner, map[string]any{"type": "amount", "amount_ars": 500})
	if second.status != http.StatusCreated {
		t.Fatalf("create second benefit = %d %v", second.status, second.body)
	}

	a := s.generate(path, "Ana", "5491100000001")
	s.generate(path, "Beto", "5491100000002")
	if redeem := s.do(http.MethodPost, "/me/business/codes/redeem", owner, map[string]any{"code": a}); redeem.status != http.StatusOK {
		t.Fatalf("redeem = %d %v", redeem.status, redeem.body)
	}

	list := s.do(http.MethodGet, "/me/business/benefits", owner, nil)
	items, _ := list.body["benefits"].([]any)
	if list.status != http.StatusOK || len(items) != 2 {
		t.Fatalf("list = %d %v", list.status, list.body)
	}
	first, _ := items[0].(map[string]any)
	next, _ := items[1].(map[string]any)
	if first["generated_this_month"] != float64(2) || first["redeemed_this_month"] != float64(1) {
		t.Errorf("first benefit = %v, want 2 generated and 1 redeemed", first)
	}
	if next["generated_this_month"] != float64(0) || next["redeemed_this_month"] != float64(0) {
		t.Errorf("benefit with no coupons = %v, want zeros", next)
	}
}

func TestOverviewShowsTheCountsOfTheBusinessOfTheSession(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	path := s.visibleBenefit("mine@example.com", 10)
	theirs := s.visibleBenefit("theirs@example.com", 10)
	owner := "token-mine@example.com"
	a := s.generate(path, "Ana", "5491100000001")
	s.generate(path, "Beto", "5491100000002")
	s.generate(theirs, "Carla", "5491100000003")
	if redeem := s.do(http.MethodPost, "/me/business/codes/redeem", owner, map[string]any{"code": a}); redeem.status != http.StatusOK {
		t.Fatalf("redeem = %d %v", redeem.status, redeem.body)
	}
	var business int64
	if err := s.pool.QueryRow(t.Context(), `SELECT b.id FROM businesses b JOIN users u ON u.id = b.owner_user_id WHERE u.email = 'mine@example.com'`).Scan(&business); err != nil {
		t.Fatalf("find business: %v", err)
	}
	if _, err := s.pool.Exec(t.Context(), `INSERT INTO metric_daily (business_id, date, metric, count) VALUES ($1, (now() AT TIME ZONE 'America/Argentina/Cordoba')::date, 'profile_view', 7)`, business); err != nil {
		t.Fatalf("insert metric: %v", err)
	}

	got := s.do(http.MethodGet, "/me/business/overview", owner, nil)
	if got.status != http.StatusOK || got.body["active_coupons"] != float64(1) || got.body["redeemed_today"] != float64(1) ||
		got.body["profile_views_7d"] != float64(7) || got.body["whatsapp_clicks_7d"] != float64(0) {
		t.Errorf("overview = %d %v, want 1 active, 1 redeemed, 7 views and 0 clicks", got.status, got.body)
	}
	if got.header.Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got.header.Get("Cache-Control"))
	}

	_, admin := s.user("admin@example.com", "super_admin")
	if got := s.do(http.MethodGet, "/me/business/overview", "", nil); got.status != http.StatusUnauthorized {
		t.Errorf("no token = %d, want 401", got.status)
	}
	if got := s.do(http.MethodGet, "/me/business/overview", admin, nil); got.status != http.StatusForbidden {
		t.Errorf("super admin = %d, want 403", got.status)
	}
}

func TestPreviewTellsHowManyCouponsTheCustomerAlreadyRedeemedHere(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	path := s.visibleBenefit("owner@example.com", 10)
	owner := "token-owner@example.com"

	first := s.generate(path, "Ana", "5491100000001")
	got := s.do(http.MethodGet, "/me/business/codes/"+first, owner, nil)
	if got.body["customer_redeemed_count"] != float64(0) {
		t.Errorf("first coupon preview = %v, want 0 redeemed before", got.body)
	}
	// No history is an empty list, never null: the panel can map over it as it is.
	if list, ok := got.body["previous_redemptions"].([]any); !ok || len(list) != 0 {
		t.Errorf("first coupon previous_redemptions = %#v, want []", got.body["previous_redemptions"])
	}
	s.do(http.MethodPost, "/me/business/codes/redeem", owner, map[string]any{"code": first})

	second := s.generate(path, "Ana", "5491100000001")
	got = s.do(http.MethodGet, "/me/business/codes/"+second, owner, nil)
	if got.body["customer_redeemed_count"] != float64(1) {
		t.Errorf("second coupon preview = %v, want 1 redeemed before", got.body)
	}
	list, _ := got.body["previous_redemptions"].([]any)
	if len(list) != 1 {
		t.Fatalf("second coupon previous_redemptions = %v, want the one coupon redeemed before", got.body["previous_redemptions"])
	}
	past, _ := list[0].(map[string]any)
	benefit, _ := past["benefit"].(map[string]any)
	if past["redeemed_at"] == nil || past["branch_name"] == "" || past["branch_name"] == nil || benefit["short_label"] != "-15%" {
		t.Errorf("past redemption = %v, want its date, its branch and the -15%% it had", past)
	}
	if raw := fmt.Sprint(got.body); strings.Contains(raw, "5491100000001") {
		t.Errorf("preview leaks the customer's phone: %s", raw)
	}
}
