package main

import (
	"net/http"
	"testing"
	"time"

	billingdomain "github.com/joaquinBrassinne/MOVO/internal/billing/domain"
	identitydomain "github.com/joaquinBrassinne/MOVO/internal/identity/domain"
)

// TestPaymentToVisibilityEndToEnd walks the DoD of phase 5 through the real router, real
// tokens and a real database: a business pending review is not public; the super admin
// confirms the payment with a plan and a date, which activates it and makes it public;
// a monthly payment extends the date; a plan change reaches the owner's limits; and the
// business disappears once the grace is over.
func TestPaymentToVisibilityEndToEnd(t *testing.T) {
	t.Parallel()

	h := newAPIHarness(t)
	ctx := t.Context()
	ownerToken := h.user("owner@example.com", identitydomain.RoleBusinessAdmin)
	adminToken := h.user("root@example.com", identitydomain.RoleSuperAdmin)

	var owner, business int64
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(h.pool.QueryRow(ctx, `SELECT id FROM users WHERE email = 'owner@example.com'`).Scan(&owner))
	must(h.pool.QueryRow(ctx, `
INSERT INTO businesses (owner_user_id, category_id, name, slug, whatsapp_number, status)
VALUES ($1, (SELECT id FROM categories WHERE slug = 'yoga'), 'Studio', 'studio', '5491112345678', 'pending_review') RETURNING id`, owner).Scan(&business))
	_, err := h.pool.Exec(ctx, `
INSERT INTO branches (business_id, name, address, city, location)
VALUES ($1, 'Centro', 'Calle 1', 'Cordoba', ST_SetSRID(ST_MakePoint(-64.18, -31.42), 4326)::geography)`, business)
	must(err)

	base := "/admin/subscriptions/" + itoa(business)
	today := billingdomain.DayOf(time.Now())
	confirm := map[string]any{"plan": "pro", "paid_until": today.AddDays(30).String(), "mp_subscription_ref": "MP-123"}

	// The view is the only place that decides who is public (SDD 4.1).
	public := func() bool {
		t.Helper()
		var n int
		must(h.pool.QueryRow(ctx, `SELECT count(*) FROM visible_branches WHERE business_id = $1`, business).Scan(&n))
		return n > 0
	}
	if public() {
		t.Fatal("a business pending review must not be public")
	}

	// Only the super admin can touch subscriptions.
	for _, c := range []struct {
		name, method, path, token string
		body                      any
		status                    int
	}{
		{"no token", http.MethodPost, base, "", confirm, http.StatusUnauthorized},
		{"owner confirming their own payment", http.MethodPost, base, ownerToken, confirm, http.StatusForbidden},
		{"owner changing their own plan", http.MethodPut, base + "/plan", ownerToken, map[string]any{"plan": "pro"}, http.StatusForbidden},
		{"owner registering a payment", http.MethodPost, base + "/payments", ownerToken, nil, http.StatusForbidden},
		{"owner reading", http.MethodGet, base, ownerToken, nil, http.StatusForbidden},
	} {
		if status, body := h.do(c.method, c.path, c.token, c.body); status != c.status {
			t.Errorf("%s = %d %v, want %d", c.name, status, body, c.status)
		}
	}
	if public() {
		t.Fatal("rejected calls must change nothing")
	}

	// Validation: nothing is stored for a bad request.
	for _, c := range []struct {
		name string
		body map[string]any
	}{
		{"unknown plan", map[string]any{"plan": "gold", "paid_until": today.AddDays(30).String()}},
		{"missing date", map[string]any{"plan": "pro"}},
		{"date in the past", map[string]any{"plan": "pro", "paid_until": today.AddDays(-1).String()}},
	} {
		if status, body := h.do(http.MethodPost, base, adminToken, c.body); status != http.StatusUnprocessableEntity || errorCode(body) != "validation_failed" {
			t.Errorf("%s = %d %q, want 422 validation_failed", c.name, status, errorCode(body))
		}
	}
	if status, body := h.do(http.MethodGet, base, adminToken, nil); status != http.StatusNotFound || errorCode(body) != "subscription_not_found" {
		t.Errorf("no subscription yet = %d %q, want 404 subscription_not_found", status, errorCode(body))
	}

	// Confirming activates the business and makes it public, in one step.
	status, body := h.do(http.MethodPost, base, adminToken, confirm)
	if status != http.StatusCreated || body["plan"] != "pro" || body["status"] != "active" {
		t.Fatalf("confirm = %d %v, want 201 pro active", status, body)
	}
	var businessStatus string
	must(h.pool.QueryRow(ctx, `SELECT status FROM businesses WHERE id = $1`, business).Scan(&businessStatus))
	if businessStatus != "active" {
		t.Errorf("business status = %q, want active", businessStatus)
	}
	if !public() {
		t.Error("a business with its payment confirmed must be public")
	}
	if status, body := h.do(http.MethodPost, base, adminToken, confirm); status != http.StatusConflict || errorCode(body) != "subscription_exists" {
		t.Errorf("second confirm = %d %q, want 409 subscription_exists", status, errorCode(body))
	}

	// The plan the admin chose is the one the owner lives under.
	if status, body := h.do(http.MethodGet, "/me/plan", ownerToken, nil); status != http.StatusOK || body["plan"] != "pro" {
		t.Errorf("owner plan = %d %v, want pro", status, body)
	}

	// A monthly payment extends the date; a plan change keeps it.
	status, body = h.do(http.MethodPost, base+"/payments", adminToken, nil)
	if want := today.AddDays(30).AddMonth().String(); status != http.StatusOK || body["paid_until"] != want {
		t.Errorf("register payment = %d paid_until %v, want 200 %s", status, body["paid_until"], want)
	}
	// Two more branches make three: Starter allows two, so the newest is switched off.
	for _, name := range []string{"Norte", "Sur"} {
		_, err = h.pool.Exec(ctx, `
INSERT INTO branches (business_id, name, address, city, location)
VALUES ($1, $2, 'Calle 2', 'Cordoba', ST_SetSRID(ST_MakePoint(-64.19, -31.43), 4326)::geography)`, business, name)
		must(err)
	}
	status, body = h.do(http.MethodPut, base+"/plan", adminToken, map[string]any{"plan": "starter"})
	if status != http.StatusOK || body["plan"] != "starter" {
		t.Errorf("change plan = %d %v, want 200 starter", status, body)
	}
	if off, _ := body["deactivated"].(map[string]any)["branches"].([]any); len(off) != 1 || off[0].(map[string]any)["name"] != "Sur" {
		t.Errorf("deactivated = %v, want the newest branch (Sur)", body["deactivated"])
	}
	// The owner is told, can dismiss it, and nobody else sees it.
	status, body = h.do(http.MethodGet, "/me/alerts", ownerToken, nil)
	alertList, _ := body["alerts"].([]any)
	if status != http.StatusOK || len(alertList) != 1 || alertList[0].(map[string]any)["type"] != "plan_downgrade" {
		t.Fatalf("owner alerts = %d %v, want one plan_downgrade", status, body)
	}
	alertID := itoa(int64(alertList[0].(map[string]any)["id"].(float64)))
	if status, _ := h.do(http.MethodGet, "/me/alerts", "", nil); status != http.StatusUnauthorized {
		t.Errorf("alerts without a token = %d, want 401", status)
	}
	if status, _ := h.do(http.MethodGet, "/me/alerts", adminToken, nil); status != http.StatusForbidden {
		t.Errorf("alerts as the admin = %d, want 403", status)
	}
	if status, _ := h.do(http.MethodPost, "/me/alerts/"+alertID+"/read", ownerToken, nil); status != http.StatusNoContent {
		t.Errorf("dismiss = %d, want 204", status)
	}
	if status, body := h.do(http.MethodGet, "/me/alerts", ownerToken, nil); status != http.StatusOK || len(body["alerts"].([]any)) != 0 {
		t.Errorf("alerts after dismissing = %d %v, want none", status, body)
	}
	if status, body := h.do(http.MethodPut, base+"/plan", adminToken, map[string]any{"plan": "starter"}); status != http.StatusConflict || errorCode(body) != "no_change" {
		t.Errorf("same plan = %d %q, want 409 no_change", status, errorCode(body))
	}
	if status, body := h.do(http.MethodGet, "/me/plan", ownerToken, nil); status != http.StatusOK || body["plan"] != "starter" {
		t.Errorf("owner plan after the change = %d %v, want starter", status, body)
	}

	// The "por vencer" view lists it once it is about to end, with the reminder link.
	_, err = h.pool.Exec(ctx, `UPDATE subscriptions SET paid_until = CURRENT_DATE + 3 WHERE business_id = $1`, business)
	must(err)
	if status, body := h.do(http.MethodGet, "/admin/subscriptions?status=expiring", adminToken, nil); status != http.StatusOK {
		t.Errorf("expiring list = %d %v, want 200", status, body)
	} else if list, _ := body["subscriptions"].([]any); len(list) != 1 || list[0].(map[string]any)["reminder_url"] == "" {
		t.Errorf("expiring list = %v, want the business with a reminder link", body)
	}
	if status, _ := h.do(http.MethodGet, "/admin/subscriptions", ownerToken, nil); status != http.StatusForbidden {
		t.Errorf("an owner reading the list = %d, want 403", status)
	}
	if status, body := h.do(http.MethodGet, "/admin/subscriptions?status=bogus", adminToken, nil); status != http.StatusUnprocessableEntity || errorCode(body) != "validation_failed" {
		t.Errorf("bad filter = %d %q, want 422 validation_failed", status, errorCode(body))
	}

	// The three actions are in the history of the business, with who did them.
	var actions []string
	rows, err := h.pool.Query(ctx, `SELECT action FROM audit_log WHERE entity = 'business' AND entity_id = $1 ORDER BY id`, business)
	must(err)
	defer rows.Close()
	for rows.Next() {
		var action string
		must(rows.Scan(&action))
		actions = append(actions, action)
	}
	want := []string{"billing.confirm_payment", "billing.register_payment", "billing.change_plan"}
	if len(actions) != len(want) {
		t.Fatalf("audit actions = %v, want %v", actions, want)
	}
	for i := range want {
		if actions[i] != want[i] {
			t.Errorf("audit actions = %v, want %v", actions, want)
			break
		}
	}

	// Once paid_until plus the grace is over, the business is gone from the public.
	_, err = h.pool.Exec(ctx, `UPDATE subscriptions SET paid_until = CURRENT_DATE - 30 WHERE business_id = $1`, business)
	must(err)
	if public() {
		t.Error("a business past its grace must not be public")
	}
}
