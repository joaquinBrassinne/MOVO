package main

import (
	"net/http"
	"testing"

	identitydomain "github.com/joaquinBrassinne/MOVO/internal/identity/domain"
)

// TestOwnerMetricsFollowThePeriodAndOnlyShowTheirOwnBusiness seeds views and coupons inside
// and outside each window and checks the figures the owner reads for 7, 30 and 90 days.
func TestOwnerMetricsFollowThePeriodAndOnlyShowTheirOwnBusiness(t *testing.T) {
	t.Parallel()

	h := newAPIHarness(t)
	ctx := t.Context()
	ownerToken := h.user("owner@example.com", identitydomain.RoleBusinessAdmin)
	otherToken := h.user("other@example.com", identitydomain.RoleBusinessAdmin)
	adminToken := h.user("root@example.com", identitydomain.RoleSuperAdmin)
	strangerToken := h.user("stranger@example.com", identitydomain.RoleBusinessAdmin)

	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	seed := func(email, name, slug string) (owner, business, branch, benefit int64) {
		must(h.pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&owner))
		must(h.pool.QueryRow(ctx, `
INSERT INTO businesses (owner_user_id, category_id, name, slug, whatsapp_number, status)
VALUES ($1, (SELECT id FROM categories WHERE slug = 'yoga'), $2, $3, '5491112345678', 'active') RETURNING id`, owner, name, slug).Scan(&business))
		must(h.pool.QueryRow(ctx, `
INSERT INTO branches (business_id, name, address, city, location)
VALUES ($1, 'Centro', 'Calle 1', 'Cordoba', ST_SetSRID(ST_MakePoint(-64.18, -31.42), 4326)::geography) RETURNING id`, business).Scan(&branch))
		must(h.pool.QueryRow(ctx, `INSERT INTO benefits (business_id, type, percent) VALUES ($1, 'percent', 10) RETURNING id`, business).Scan(&benefit))
		return owner, business, branch, benefit
	}
	owner, business, branch, benefit := seed("owner@example.com", "Studio", "studio")
	_, otherBusiness, _, _ := seed("other@example.com", "Other", "other")
	_ = otherToken

	var c1, c2 int64
	must(h.pool.QueryRow(ctx, `INSERT INTO customers (name, phone) VALUES ('Ana', '5491187654321') RETURNING id`).Scan(&c1))
	must(h.pool.QueryRow(ctx, `INSERT INTO customers (name, phone) VALUES ('Beto', '5491187654322') RETURNING id`).Scan(&c2))

	// Views: 5 today, 3 eight days ago, 100 forty days ago; and 999 for the other business.
	view := func(biz int64, daysAgo, n int) {
		t.Helper()
		_, err := h.pool.Exec(ctx, `
INSERT INTO metric_daily (business_id, date, metric, count)
VALUES ($1, (now() AT TIME ZONE 'America/Argentina/Cordoba')::date - $2::int, 'profile_view', $3)`, biz, daysAgo, n)
		must(err)
	}
	view(business, 0, 5)
	view(business, 8, 3)
	view(business, 40, 100)
	view(otherBusiness, 0, 999)

	// Coupons: one pending and one redeemed today, one pending and one redeemed forty days ago.
	coupon := func(code string, customer int64, status string, daysAgo int) {
		t.Helper()
		hash := make([]byte, 32)
		copy(hash, code)
		if status == "redeemed" {
			_, err := h.pool.Exec(ctx, `
INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, status, expires_at, created_at, redeemed_at, redeemed_by, redeemed_branch_id)
VALUES ($1, $2, $3, $4, $5, 'redeemed', now() + interval '48 hours', now() - $6 * interval '1 day', now() - $6 * interval '1 day', $7, $8)`,
				code, benefit, business, customer, hash, daysAgo, owner, branch)
			must(err)
			return
		}
		_, err := h.pool.Exec(ctx, `
INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, status, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5, 'pending', now() + interval '48 hours', now() - $6 * interval '1 day')`,
			code, benefit, business, customer, hash, daysAgo)
		must(err)
	}
	coupon("AAAAAAA2", c1, "pending", 0)
	coupon("AAAAAAA3", c1, "redeemed", 0)
	coupon("AAAAAAA4", c2, "pending", 40)
	coupon("AAAAAAA5", c2, "redeemed", 40)

	for _, tc := range []struct {
		path                  string
		days                  float64
		views, generated, red float64
	}{
		{"/me/business/metrics?days=7", 7, 5, 2, 1},
		{"/me/business/metrics", 30, 8, 2, 1},
		{"/me/business/metrics?days=90", 90, 108, 4, 2},
	} {
		status, body := h.do(http.MethodGet, tc.path, ownerToken, nil)
		period, _ := body["period"].(map[string]any)
		coupons, _ := body["coupons"].(map[string]any)
		if status != http.StatusOK || period["days"] != tc.days || body["profile_views"] != tc.views ||
			coupons["generated"] != tc.generated || coupons["redeemed"] != tc.red {
			t.Errorf("%s = %d %v, want %v days, %v views, coupons %v/%v", tc.path, status, body, tc.days, tc.views, tc.generated, tc.red)
		}
	}

	if status, body := h.do(http.MethodGet, "/me/business/metrics?days=14", ownerToken, nil); status != http.StatusUnprocessableEntity || errorCode(body) != "validation_failed" {
		t.Errorf("days=14 = %d %q, want 422 validation_failed", status, errorCode(body))
	}
	if status, body := h.do(http.MethodGet, "/me/business/metrics?days=abc", ownerToken, nil); status != http.StatusUnprocessableEntity || errorCode(body) != "validation_failed" {
		t.Errorf("days=abc = %d %q, want 422 validation_failed", status, errorCode(body))
	}
	if status, _ := h.do(http.MethodGet, "/me/business/metrics", "", nil); status != http.StatusUnauthorized {
		t.Errorf("anonymous = %d, want 401", status)
	}
	if status, _ := h.do(http.MethodGet, "/me/business/metrics", adminToken, nil); status != http.StatusForbidden {
		t.Errorf("super admin = %d, want 403", status)
	}
	if status, body := h.do(http.MethodGet, "/me/business/metrics", strangerToken, nil); status != http.StatusNotFound || errorCode(body) != "business_not_found" {
		t.Errorf("owner without a business = %d %q, want 404 business_not_found", status, errorCode(body))
	}
}
