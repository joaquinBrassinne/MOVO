package benefitshttp_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestOwnerSeesTheRecentActivityOfTheirBusinessOnly(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	mine := s.visibleBenefit("mine@example.com", 5)
	theirs := s.visibleBenefit("theirs@example.com", 5)
	owner := "token-mine@example.com"
	code := s.generate(mine, "Ana", "5491112345678")
	s.generate(theirs, "Beto", "5491187654321")
	if redeem := s.do(http.MethodPost, "/me/business/codes/redeem", owner, map[string]any{"code": code}); redeem.status != http.StatusOK {
		t.Fatalf("redeem = %d %v", redeem.status, redeem.body)
	}

	got := s.do(http.MethodGet, "/me/business/activity?since="+url.QueryEscape("2000-01-01T00:00:00Z"), owner, nil)
	items, _ := got.body["items"].([]any)
	if got.status != http.StatusOK || len(items) != 2 || got.body["new_count"] != float64(2) {
		t.Fatalf("activity = %d %v, want two items and two news", got.status, got.body)
	}
	newest, _ := items[0].(map[string]any)
	oldest, _ := items[1].(map[string]any)
	if newest["type"] != "coupon_redeemed" || oldest["type"] != "coupon_generated" || newest["customer_name"] != "Ana" || newest["at"] == nil {
		t.Errorf("items = %v", items)
	}
	if got.header.Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got.header.Get("Cache-Control"))
	}
	raw, _ := json.Marshal(got.body)
	for _, leak := range []string{"5491112345678", "5491187654321", "Beto", "phone"} {
		if strings.Contains(string(raw), leak) {
			t.Errorf("body %s leaks %q", raw, leak)
		}
	}

	// Without a mark everything is already seen; a mark in the future leaves nothing new.
	if none := s.do(http.MethodGet, "/me/business/activity", owner, nil); none.body["new_count"] != float64(0) || len(none.body["items"].([]any)) != 2 {
		t.Errorf("no mark = %v", none.body)
	}
	if later := s.do(http.MethodGet, "/me/business/activity?since="+url.QueryEscape("2999-01-01T00:00:00Z"), owner, nil); later.body["new_count"] != float64(0) {
		t.Errorf("future mark = %v", later.body)
	}
}

func TestActivityRejectsABadMarkAndNeedsAnOwner(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	s.visibleBenefit("owner@example.com", 5)
	_, admin := s.user("admin@example.com", "super_admin")

	if got := s.do(http.MethodGet, "/me/business/activity?since=yesterday", "token-owner@example.com", nil); got.status != http.StatusUnprocessableEntity || got.errorField() != "since" {
		t.Errorf("bad mark = %d field %q, want 422 on since", got.status, got.errorField())
	}
	if got := s.do(http.MethodGet, "/me/business/activity", "", nil); got.status != http.StatusUnauthorized {
		t.Errorf("no token = %d, want 401", got.status)
	}
	if got := s.do(http.MethodGet, "/me/business/activity", admin, nil); got.status != http.StatusForbidden {
		t.Errorf("super admin = %d, want 403", got.status)
	}
}
