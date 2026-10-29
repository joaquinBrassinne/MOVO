package main

import (
	"crypto/rand"
	"net/http"
	"testing"

	benefitsdomain "github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
	identitydomain "github.com/joaquinBrassinne/MOVO/internal/identity/domain"
)

// TestReviewFormOpensWithATokenTheCouponsModuleIssued pins that the reviews routes find a
// coupon by the token the benefits module generated and hashed, the one a customer holds in
// their link: the form shows where it was redeemed and the review can be written with it.
func TestReviewFormOpensWithATokenTheCouponsModuleIssued(t *testing.T) {
	t.Parallel()

	h := newAPIHarness(t)
	ctx := t.Context()
	_ = h.user("owner@example.com", identitydomain.RoleBusinessAdmin)

	var owner, business, branch, benefit, customer int64
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(h.pool.QueryRow(ctx, `SELECT id FROM users WHERE email = 'owner@example.com'`).Scan(&owner))
	must(h.pool.QueryRow(ctx, `
INSERT INTO businesses (owner_user_id, category_id, name, slug, whatsapp_number, status)
VALUES ($1, (SELECT id FROM categories WHERE slug = 'yoga'), 'Studio', 'studio', '5491112345678', 'active') RETURNING id`, owner).Scan(&business))
	must(h.pool.QueryRow(ctx, `
INSERT INTO branches (business_id, name, address, city, location)
VALUES ($1, 'Centro', 'Calle 1', 'Cordoba', ST_SetSRID(ST_MakePoint(-64.18, -31.42), 4326)::geography) RETURNING id`, business).Scan(&branch))
	must(h.pool.QueryRow(ctx, `INSERT INTO benefits (business_id, type, percent) VALUES ($1, 'percent', 10) RETURNING id`, business).Scan(&benefit))
	must(h.pool.QueryRow(ctx, `INSERT INTO customers (name, phone) VALUES ('Lucía Martínez', '5491187654321') RETURNING id`).Scan(&customer))

	token, hash, err := benefitsdomain.NewAccessToken(rand.Reader)
	must(err)
	_, err = h.pool.Exec(ctx, `
INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, status, expires_at, redeemed_at, redeemed_by, redeemed_branch_id)
VALUES ('ABCDEFGH', $1, $2, $3, $4, 'redeemed', now() + interval '48 hours', now(), $5, $6)`,
		benefit, business, customer, hash, owner, branch)
	must(err)

	status, body := h.do(http.MethodGet, "/access/"+token+"/review", "", nil)
	if status != http.StatusOK || body["state"] != "open" || body["branch_name"] != "Centro" || body["display_name"] != "Lucía M." {
		t.Fatalf("form = %d %v, want 200, open, the branch and the name to show", status, body)
	}
	if status, body := h.do(http.MethodPost, "/access/"+token+"/reviews", "", map[string]any{"rating": 5}); status != http.StatusCreated {
		t.Fatalf("write review = %d %v, want 201", status, body)
	}
	if status, body := h.do(http.MethodGet, "/access/"+token+"/review", "", nil); status != http.StatusOK || body["state"] != "reviewed" {
		t.Errorf("form after reviewing = %d %v, want reviewed", status, body)
	}
}
