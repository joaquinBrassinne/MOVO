package main

import (
	"net/http"
	"testing"

	identitydomain "github.com/joaquinBrassinne/MOVO/internal/identity/domain"
)

// TestMostDemandedActivitiesRankByViewsPlusClicks seeds the counters of several services and
// checks the ranking: views plus WhatsApp taps, ties by name, five at most, deleted services
// and taps with no service left out, and nothing from another business.
func TestMostDemandedActivitiesRankByViewsPlusClicks(t *testing.T) {
	t.Parallel()

	h := newAPIHarness(t)
	ctx := t.Context()
	ownerToken := h.user("owner@example.com", identitydomain.RoleBusinessAdmin)
	h.user("other@example.com", identitydomain.RoleBusinessAdmin)
	adminToken := h.user("root@example.com", identitydomain.RoleSuperAdmin)
	strangerToken := h.user("stranger@example.com", identitydomain.RoleBusinessAdmin)

	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	seed := func(email, slug string) (business, branch int64) {
		var owner int64
		must(h.pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&owner))
		must(h.pool.QueryRow(ctx, `
INSERT INTO businesses (owner_user_id, category_id, name, slug, whatsapp_number, status)
VALUES ($1, (SELECT id FROM categories WHERE slug = 'yoga'), $2, $2, '5491112345678', 'active') RETURNING id`, owner, slug).Scan(&business))
		must(h.pool.QueryRow(ctx, `
INSERT INTO branches (business_id, name, address, city, location)
VALUES ($1, 'Centro', 'Calle 1', 'Cordoba', ST_SetSRID(ST_MakePoint(-64.18, -31.42), 4326)::geography) RETURNING id`, business).Scan(&branch))
		return business, branch
	}
	business, branch := seed("owner@example.com", "studio")
	otherBusiness, otherBranch := seed("other@example.com", "other")

	service := func(biz int64, name string, deleted bool) int64 {
		var id int64
		must(h.pool.QueryRow(ctx, `
INSERT INTO services (business_id, name, deleted_at) VALUES ($1, $2, CASE WHEN $3 THEN now() END) RETURNING id`, biz, name, deleted).Scan(&id))
		return id
	}
	count := func(biz, br, svc int64, metric string, daysAgo, n int) {
		t.Helper()
		var serviceArg any
		if svc != 0 {
			serviceArg = svc
		}
		_, err := h.pool.Exec(ctx, `
INSERT INTO metric_daily (business_id, branch_id, service_id, date, metric, count)
VALUES ($1, $2, $3, (now() AT TIME ZONE 'America/Argentina/Cordoba')::date - $4::int, $5, $6)`, biz, br, serviceArg, daysAgo, metric, n)
		must(err)
	}

	pilates := service(business, "Pilates", false) // 10 views + 5 clicks = 15
	yoga := service(business, "Yoga", false)       // 8 + 7 = 15, ties with Pilates: name decides
	boxeo := service(business, "Boxeo", false)     // 3 + 0 = 3 now, plus 50 views 40 days ago
	spinning := service(business, "Spinning", false)
	crossfit := service(business, "Crossfit", false)
	zumba := service(business, "Zumba", false) // 1: the sixth, left out by the limit
	gone := service(business, "Borrado", true) // deleted: left out
	foreign := service(otherBusiness, "Ajeno", false)

	count(business, branch, pilates, "service_view", 0, 10)
	count(business, branch, pilates, "whatsapp_click", 1, 5)
	count(business, branch, yoga, "service_view", 2, 8)
	count(business, branch, yoga, "whatsapp_click", 2, 7)
	count(business, branch, boxeo, "service_view", 0, 3)
	count(business, branch, boxeo, "service_view", 40, 50)
	count(business, branch, spinning, "service_view", 0, 9)
	count(business, branch, crossfit, "whatsapp_click", 0, 4)
	count(business, branch, zumba, "service_view", 0, 1)
	count(business, branch, gone, "service_view", 0, 999)
	count(business, branch, 0, "whatsapp_click", 0, 999) // a tap with no service is not an activity
	count(otherBusiness, otherBranch, foreign, "service_view", 0, 999)

	status, body := h.do(http.MethodGet, "/me/business/metrics/activities", ownerToken, nil)
	list, _ := body["activities"].([]any)
	if status != http.StatusOK || len(list) != 5 {
		t.Fatalf("activities = %d %v, want 200 with five", status, body)
	}
	var names []string
	for _, item := range list {
		names = append(names, item.(map[string]any)["name"].(string))
	}
	want := []string{"Pilates", "Yoga", "Spinning", "Crossfit", "Boxeo"}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("ranking = %v, want %v", names, want)
		}
	}
	first := list[0].(map[string]any)
	if first["views"] != float64(10) || first["whatsapp_clicks"] != float64(5) || first["service_id"] != float64(pilates) {
		t.Errorf("first = %v, want 10 views, 5 clicks, service %d", first, pilates)
	}

	// Boxeo's 50 old views only count once the window reaches them.
	_, body = h.do(http.MethodGet, "/me/business/metrics/activities?period=90", ownerToken, nil)
	if list, _ := body["activities"].([]any); len(list) == 0 || list[0].(map[string]any)["name"] != "Boxeo" {
		t.Errorf("90 days = %v, want Boxeo on top with its 53 views", body)
	}
	_, body = h.do(http.MethodGet, "/me/business/metrics/activities?period=7", ownerToken, nil)
	if list, _ := body["activities"].([]any); len(list) != 5 || list[0].(map[string]any)["name"] != "Pilates" {
		t.Errorf("7 days = %v, want Pilates first of five", body)
	}

	if status, body := h.do(http.MethodGet, "/me/business/metrics/activities?period=14", ownerToken, nil); status != http.StatusUnprocessableEntity || errorCode(body) != "validation_failed" {
		t.Errorf("period=14 = %d %q, want 422 validation_failed", status, errorCode(body))
	}
	if status, _ := h.do(http.MethodGet, "/me/business/metrics/activities", "", nil); status != http.StatusUnauthorized {
		t.Errorf("anonymous = %d, want 401", status)
	}
	if status, _ := h.do(http.MethodGet, "/me/business/metrics/activities", adminToken, nil); status != http.StatusForbidden {
		t.Errorf("super admin = %d, want 403", status)
	}
	if status, body := h.do(http.MethodGet, "/me/business/metrics/activities", strangerToken, nil); status != http.StatusNotFound || errorCode(body) != "business_not_found" {
		t.Errorf("owner without a business = %d %q, want 404 business_not_found", status, errorCode(body))
	}
}
