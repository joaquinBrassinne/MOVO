package main

import (
	"net/http"
	"strconv"
	"testing"

	identitydomain "github.com/joaquinBrassinne/MOVO/internal/identity/domain"
	reviewsdomain "github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

// TestReviewLifeCycleEndToEnd walks a review through the real router, the real tokens and
// a real database: the customer writes it with the token of their coupon, the public
// reads it, the owner reports it, and the super admin hides it. It also proves the
// reviews routes do not collide with the business ones under /me/business and /admin.
func TestReviewLifeCycleEndToEnd(t *testing.T) {
	t.Parallel()

	h := newAPIHarness(t)
	ctx := t.Context()
	ownerToken := h.user("owner@example.com", identitydomain.RoleBusinessAdmin)
	strangerToken := h.user("stranger@example.com", identitydomain.RoleBusinessAdmin)
	adminToken := h.user("root@example.com", identitydomain.RoleSuperAdmin)

	var owner, business, branch, benefit, customer int64
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(h.pool.QueryRow(ctx, `SELECT id FROM users WHERE email = 'owner@example.com'`).Scan(&owner))
	must(h.pool.QueryRow(ctx, `
INSERT INTO businesses (owner_user_id, category_id, name, slug, whatsapp_number, status, pilot)
VALUES ($1, (SELECT id FROM categories WHERE slug = 'yoga'), 'Studio', 'studio', '5491112345678', 'active', true) RETURNING id`, owner).Scan(&business))
	must(h.pool.QueryRow(ctx, `
INSERT INTO branches (business_id, name, address, city, location)
VALUES ($1, 'Centro', 'Calle 1', 'Cordoba', ST_SetSRID(ST_MakePoint(-64.18, -31.42), 4326)::geography) RETURNING id`, business).Scan(&branch))
	must(h.pool.QueryRow(ctx, `INSERT INTO benefits (business_id, type, percent) VALUES ($1, 'percent', 10) RETURNING id`, business).Scan(&benefit))
	must(h.pool.QueryRow(ctx, `INSERT INTO customers (name, phone) VALUES ('Ana García', '5491187654321') RETURNING id`).Scan(&customer))
	const accessToken = "the-customers-access-token"
	_, err := h.pool.Exec(ctx, `
INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, status, expires_at, redeemed_at, redeemed_by, redeemed_branch_id)
VALUES ('ABCDEFGH', $1, $2, $3, $4, 'redeemed', now() + interval '48 hours', now(), $5, $6)`,
		benefit, business, customer, reviewsdomain.HashAccessToken(accessToken), owner, branch)
	must(err)

	// The customer reviews with the token of the coupon, with no account.
	if status, body := h.do(http.MethodPost, "/access/"+accessToken+"/reviews", "", map[string]any{"rating": 5, "comment": "Muy buena clase"}); status != http.StatusCreated {
		t.Fatalf("write review = %d %v, want 201", status, body)
	}
	var reviewID int64
	must(h.pool.QueryRow(ctx, `SELECT id FROM reviews`).Scan(&reviewID))

	// The public reads it.
	listPath := "/branches/" + itoa(branch) + "/reviews"
	if status, body := h.do(http.MethodGet, listPath, "", nil); status != http.StatusOK || len(body["reviews"].([]any)) != 1 {
		t.Fatalf("public list = %d %v, want one review", status, body)
	}

	// Each role reaches only its own route, and a business without reviews of its own
	// learns nothing about this one.
	reportPath := "/me/business/reviews/" + itoa(reviewID) + "/report"
	hidePath := "/admin/reviews/" + itoa(reviewID) + "/status"
	for _, c := range []struct {
		name, method, path, token string
		body                      any
		status                    int
		code                      string
	}{
		{"report without a token", http.MethodPost, reportPath, "", nil, http.StatusUnauthorized, "unauthorized"},
		{"the admin cannot report as an owner", http.MethodPost, reportPath, adminToken, nil, http.StatusForbidden, "forbidden"},
		{"an owner without a business cannot report", http.MethodPost, reportPath, strangerToken, nil, http.StatusNotFound, "business_not_found"},
		{"an owner cannot hide", http.MethodPut, hidePath, ownerToken, map[string]any{"status": "hidden"}, http.StatusForbidden, "forbidden"},
		{"hide without a token", http.MethodPut, hidePath, "", map[string]any{"status": "hidden"}, http.StatusUnauthorized, "unauthorized"},
		{"the admin cannot report a status the owner owns", http.MethodPut, hidePath, adminToken, map[string]any{"status": "reported"}, http.StatusUnprocessableEntity, "validation_failed"},
	} {
		status, body := h.do(c.method, c.path, c.token, c.body)
		if status != c.status || errorCode(body) != c.code {
			t.Errorf("%s = %d %q, want %d %q", c.name, status, errorCode(body), c.status, c.code)
		}
	}

	// The owner finds the review, with the id the report needs, in their own list.
	ids := func(token, path string) []any {
		status, body := h.do(http.MethodGet, path, token, nil)
		if status != http.StatusOK {
			t.Fatalf("GET %s = %d %v, want 200", path, status, body)
		}
		var found []any
		for _, r := range body["reviews"].([]any) {
			found = append(found, r.(map[string]any)["id"])
		}
		return found
	}
	if got := ids(ownerToken, "/me/business/reviews"); len(got) != 1 || got[0] != float64(reviewID) {
		t.Errorf("owner list = %v, want the review %d", got, reviewID)
	}
	if status, _ := h.do(http.MethodGet, "/admin/reviews", ownerToken, nil); status != http.StatusForbidden {
		t.Errorf("an owner reading the queue = %d, want 403", status)
	}
	if got := ids(adminToken, "/admin/reviews"); len(got) != 0 {
		t.Errorf("queue before the report = %v, want it empty", got)
	}

	// The owner reports it: it stays public and still counts.
	if status, body := h.do(http.MethodPost, reportPath, ownerToken, nil); status != http.StatusOK || body["status"] != "reported" {
		t.Fatalf("report = %d %v, want 200 reported", status, body)
	}
	if got := ids(adminToken, "/admin/reviews"); len(got) != 1 || got[0] != float64(reviewID) {
		t.Errorf("queue after the report = %v, want the review %d", got, reviewID)
	}
	if status, body := h.do(http.MethodGet, listPath, "", nil); status != http.StatusOK || len(body["reviews"].([]any)) != 1 {
		t.Errorf("list after the report = %d %v, want it still public", status, body)
	}

	// The super admin hides it: it leaves the list and the rating.
	if status, body := h.do(http.MethodPut, hidePath, adminToken, map[string]any{"status": "hidden"}); status != http.StatusOK || body["status"] != "hidden" {
		t.Fatalf("hide = %d %v, want 200 hidden", status, body)
	}
	if status, body := h.do(http.MethodGet, listPath, "", nil); status != http.StatusOK || len(body["reviews"].([]any)) != 0 {
		t.Errorf("list after hiding = %d %v, want it empty", status, body)
	}
	if got := ids(adminToken, "/admin/reviews"); len(got) != 0 {
		t.Errorf("reported queue after hiding = %v, want it empty", got)
	}
	if got := ids(adminToken, "/admin/reviews?status=hidden"); len(got) != 1 || got[0] != float64(reviewID) {
		t.Errorf("hidden queue = %v, want the review %d, which can still be restored", got, reviewID)
	}
	if got := ids(ownerToken, "/me/business/reviews"); len(got) != 0 {
		t.Errorf("owner list after hiding = %v, want it empty: a hidden review is not theirs to act on", got)
	}
	var count int
	must(h.pool.QueryRow(ctx, `SELECT rating_count FROM branches WHERE id = $1`, branch).Scan(&count))
	if count != 0 {
		t.Errorf("rating_count after hiding = %d, want 0", count)
	}

	var audited int
	must(h.pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE entity = 'review' AND entity_id = $1`, reviewID).Scan(&audited))
	if audited != 2 {
		t.Errorf("audit entries = %d, want 2 (report and hide)", audited)
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
