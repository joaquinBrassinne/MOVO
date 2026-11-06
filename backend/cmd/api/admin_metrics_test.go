package main

import (
	"net/http"
	"testing"
	"time"

	identitydomain "github.com/joaquinBrassinne/MOVO/internal/identity/domain"
)

// TestAdminMetricsMatchTheRealData seeds a known set of businesses and coupons and checks the
// figures the super admin reads: only active businesses count, redeemed is a subset of
// generated, and "this month" leaves out last month's coupons.
func TestAdminMetricsMatchTheRealData(t *testing.T) {
	t.Parallel()

	h := newAPIHarness(t)
	ctx := t.Context()
	ownerToken := h.user("owner@example.com", identitydomain.RoleBusinessAdmin)
	adminToken := h.user("root@example.com", identitydomain.RoleSuperAdmin)

	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	var owner, business, branch, benefit, customer int64
	must(h.pool.QueryRow(ctx, `SELECT id FROM users WHERE email = 'owner@example.com'`).Scan(&owner))
	must(h.pool.QueryRow(ctx, `
INSERT INTO businesses (owner_user_id, category_id, name, slug, whatsapp_number, status)
VALUES ($1, (SELECT id FROM categories WHERE slug = 'yoga'), 'Studio', 'studio', '5491112345678', 'active') RETURNING id`, owner).Scan(&business))
	// A second business that is not active must not count.
	other := h.user("paused@example.com", identitydomain.RoleBusinessAdmin)
	_ = other
	var pausedOwner int64
	must(h.pool.QueryRow(ctx, `SELECT id FROM users WHERE email = 'paused@example.com'`).Scan(&pausedOwner))
	_, err := h.pool.Exec(ctx, `
INSERT INTO businesses (owner_user_id, category_id, name, slug, whatsapp_number, status)
VALUES ($1, (SELECT id FROM categories WHERE slug = 'yoga'), 'Paused', 'paused', '5491112345679', 'paused')`, pausedOwner)
	must(err)
	must(h.pool.QueryRow(ctx, `
INSERT INTO branches (business_id, name, address, city, location)
VALUES ($1, 'Centro', 'Calle 1', 'Cordoba', ST_SetSRID(ST_MakePoint(-64.18, -31.42), 4326)::geography) RETURNING id`, business).Scan(&branch))
	must(h.pool.QueryRow(ctx, `INSERT INTO benefits (business_id, type, percent) VALUES ($1, 'percent', 10) RETURNING id`, business).Scan(&benefit))
	must(h.pool.QueryRow(ctx, `INSERT INTO customers (name, phone) VALUES ('Ana', '5491187654321') RETURNING id`).Scan(&customer))

	var second int64
	must(h.pool.QueryRow(ctx, `INSERT INTO customers (name, phone) VALUES ('Beto', '5491187654322') RETURNING id`).Scan(&second))
	insert := func(code, status, createdAt string, customer int64) {
		t.Helper()
		hash := make([]byte, 32)
		copy(hash, code)
		if status == "redeemed" {
			_, err := h.pool.Exec(ctx, `
INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, status, expires_at, created_at, redeemed_at, redeemed_by, redeemed_branch_id)
VALUES ($1, $2, $3, $4, $5, 'redeemed', now() + interval '48 hours', $6::timestamptz, $6::timestamptz, $7, $8)`,
				code, benefit, business, customer, hash, createdAt, owner, branch)
			must(err)
			return
		}
		_, err := h.pool.Exec(ctx, `
INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, status, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6, now() + interval '48 hours', $7::timestamptz)`,
			code, benefit, business, customer, hash, status, createdAt)
		must(err)
	}
	// Two this month (one redeemed), one pending last year, one redeemed last year.
	insert("AAAAAAA2", "pending", time.Now().Format(time.RFC3339), customer)
	insert("AAAAAAA3", "redeemed", time.Now().Format(time.RFC3339), customer)
	insert("AAAAAAA4", "pending", "2020-01-15T12:00:00Z", second)
	insert("AAAAAAA5", "redeemed", "2020-01-15T12:00:00Z", second)

	status, body := h.do(http.MethodGet, "/admin/metrics", adminToken, nil)
	if status != http.StatusOK {
		t.Fatalf("metrics = %d %v, want 200", status, body)
	}
	coupons, _ := body["coupons"].(map[string]any)
	month, _ := body["coupons_this_month"].(map[string]any)
	if body["active_businesses"] != float64(1) ||
		coupons["generated"] != float64(4) || coupons["redeemed"] != float64(2) ||
		month["generated"] != float64(2) || month["redeemed"] != float64(1) {
		t.Errorf("metrics = %v, want 1 active, coupons 4/2, this month 2/1", body)
	}
	if s, _ := body["month_start"].(string); len(s) != 10 || s[8:] != "01" {
		t.Errorf("month_start = %v, want the first day of a month", body["month_start"])
	}

	if status, _ := h.do(http.MethodGet, "/admin/metrics", "", nil); status != http.StatusUnauthorized {
		t.Errorf("anonymous = %d, want 401", status)
	}
	if status, _ := h.do(http.MethodGet, "/admin/metrics", ownerToken, nil); status != http.StatusForbidden {
		t.Errorf("owner = %d, want 403", status)
	}
}
