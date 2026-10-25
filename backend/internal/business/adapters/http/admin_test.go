package businesshttp_test

import (
	"net/http"
	"strings"
	"testing"
)

func adminAction(s *server, token, id, action string) reply {
	s.t.Helper()

	return s.do(http.MethodPost, "/admin/businesses/"+id+"/actions", token, map[string]any{"action": action})
}

func TestAdminRoutesRequireASuperAdmin(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	owner := ownerWithBusiness(s, "owner@example.com")

	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/admin/businesses"},
		{http.MethodPost, "/admin/businesses/1/actions"},
	} {
		if got := s.do(route.method, route.path, "", nil); got.status != http.StatusUnauthorized {
			t.Errorf("%s %s without token = %d, want 401", route.method, route.path, got.status)
		}
		if got := s.do(route.method, route.path, owner, map[string]any{"action": "approve"}); got.status != http.StatusForbidden {
			t.Errorf("%s %s as an owner = %d, want 403", route.method, route.path, got.status)
		}
	}

	mine := s.do(http.MethodGet, "/me/business", owner, nil)
	if b, _ := mine.body["business"].(map[string]any); b["status"] != "draft" {
		t.Errorf("an owner's attempt must not change anything, status = %v", b["status"])
	}
}

// TestOwnerCreatesAndAdminActivatesAndVerifies is the phase 2 definition of done:
// an owner creates the business, the admin activates it and marks it verified,
// and it ends up active.
func TestOwnerCreatesAndAdminActivatesAndVerifies(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	owner := ownerWithBusiness(s, "owner@example.com")
	adminID, admin := s.user("root@example.com", "super_admin")

	if got := s.do(http.MethodPost, "/me/business/submit", owner, payerBody("ana@example.com", "Ana Pérez", map[string]string{"plan": "pro", "status": "approved"})); got.status != http.StatusOK {
		t.Fatalf("submit = %d %v", got.status, got.body)
	}

	list := s.do(http.MethodGet, "/admin/businesses?status=pending_review", admin, nil)
	rows, _ := list.body["businesses"].([]any)
	if list.status != http.StatusOK || len(rows) != 1 || list.body["total"] != float64(1) {
		t.Fatalf("review queue = %d %v", list.status, list.body)
	}
	row, _ := rows[0].(map[string]any)
	if row["owner_email"] != "owner@example.com" || row["category_name"] != "Yoga" || row["mp_payer_email"] != "ana@example.com" || row["mp_payer_name"] != "Ana Pérez" || row["mp_plan_hint"] != "pro" || row["branch_count"] != float64(0) {
		t.Errorf("row = %v", row)
	}
	id := itoa(int64(row["id"].(float64)))

	approved := adminAction(s, admin, id, "approve")
	if approved.status != http.StatusOK || approved.body["status"] != "active" {
		t.Fatalf("approve = %d %v", approved.status, approved.body)
	}
	verified := adminAction(s, admin, id, "verify")
	if verified.status != http.StatusOK || verified.body["verified"] != true || verified.body["status"] != "active" {
		t.Fatalf("verify = %d %v", verified.status, verified.body)
	}

	mine := s.do(http.MethodGet, "/me/business", owner, nil)
	if b, _ := mine.body["business"].(map[string]any); b["status"] != "active" || b["verified"] != true {
		t.Errorf("the owner sees %v, want active and verified", b)
	}

	var actions []string
	rowsAudit, err := s.pool.Query(t.Context(), `SELECT action FROM audit_log WHERE actor_id = $1 ORDER BY id`, adminID)
	if err != nil {
		t.Fatalf("read audit: %v", err)
	}
	defer rowsAudit.Close()
	for rowsAudit.Next() {
		var action string
		if err := rowsAudit.Scan(&action); err != nil {
			t.Fatalf("scan: %v", err)
		}
		actions = append(actions, action)
	}
	if len(actions) != 2 || actions[0] != "business.approve" || actions[1] != "business.verify" {
		t.Errorf("audit actions = %v, want the approval and the verification attributed to the admin", actions)
	}
}

func TestAdminActionRejections(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	ownerWithBusiness(s, "owner@example.com")
	_, admin := s.user("root@example.com", "super_admin")
	list := s.do(http.MethodGet, "/admin/businesses", admin, nil)
	rows, _ := list.body["businesses"].([]any)
	id := itoa(int64(rows[0].(map[string]any)["id"].(float64)))

	tests := []struct {
		name     string
		path     string
		body     any
		wantCode int
		want     string
	}{
		{"approve a draft", id, map[string]any{"action": "approve"}, http.StatusConflict, "invalid_state"},
		{"unknown action", id, map[string]any{"action": "explode"}, http.StatusUnprocessableEntity, "validation_failed"},
		{"business that does not exist", "99999", map[string]any{"action": "approve"}, http.StatusNotFound, "business_not_found"},
		{"not a number", "abc", map[string]any{"action": "approve"}, http.StatusNotFound, "business_not_found"},
		{"unknown field", id, map[string]any{"action": "approve", "force": true}, http.StatusBadRequest, "invalid_request"},
		{"unmark a regular business", id, map[string]any{"action": "unmark_pilot"}, http.StatusConflict, "no_change"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.do(http.MethodPost, "/admin/businesses/"+tt.path+"/actions", admin, tt.body)
			if got.status != tt.wantCode || got.errorCode() != tt.want {
				t.Fatalf("got %d %q, want %d %q (%v)", got.status, got.errorCode(), tt.wantCode, tt.want, got.body)
			}
		})
	}

	var n int
	if err := s.pool.QueryRow(t.Context(), `SELECT count(*) FROM audit_log`).Scan(&n); err != nil || n != 0 {
		t.Errorf("audit rows = %d, %v; rejected actions must not be logged", n, err)
	}
}

func TestAdminListFiltersAndPages(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	ownerWithBusiness(s, "a@example.com")
	ownerWithBusiness(s, "b@example.com")
	_, admin := s.user("root@example.com", "super_admin")

	all := s.do(http.MethodGet, "/admin/businesses", admin, nil)
	if rows, _ := all.body["businesses"].([]any); len(rows) != 2 || all.body["total"] != float64(2) {
		t.Fatalf("all = %v", all.body)
	}
	paged := s.do(http.MethodGet, "/admin/businesses?limit=1&offset=1", admin, nil)
	if rows, _ := paged.body["businesses"].([]any); len(rows) != 1 || paged.body["total"] != float64(2) {
		t.Errorf("second page = %v, want one row and the total of 2", paged.body)
	}
	empty := s.do(http.MethodGet, "/admin/businesses?status=paused", admin, nil)
	if rows, ok := empty.body["businesses"].([]any); !ok || len(rows) != 0 {
		t.Errorf("paused = %v, want an empty list and not null", empty.body)
	}

	for _, query := range []string{"?status=bogus", "?limit=abc", "?limit=1000", "?offset=-1"} {
		if got := s.do(http.MethodGet, "/admin/businesses"+query, admin, nil); got.status != http.StatusUnprocessableEntity || got.errorField() != "filter" {
			t.Errorf("%s = %d field %q, want 422 filter", query, got.status, got.errorField())
		}
	}
}

func TestAdminListSearchesFiltersByCategoryAndCountsByStatus(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	_, root := s.user("root@example.com", "super_admin")
	ownerWithBusiness(s, "ana@alma.example")

	other := ownerWithBusiness(s, "bruno@flow.example")
	s.do(http.MethodPost, "/me/business/submit", other, payerBody("bruno@flow.example", "Bruno Díaz", nil))

	byEmail := s.do(http.MethodGet, "/admin/businesses?q=bruno", root, nil)
	if rows, _ := byEmail.body["businesses"].([]any); len(rows) != 1 || byEmail.body["total"] != float64(1) {
		t.Fatalf("search by owner email = %v", byEmail.body)
	}

	counts, _ := byEmail.body["counts"].(map[string]any)
	if counts["draft"] != float64(1) || counts["pending_review"] != float64(1) {
		t.Errorf("counts = %v, want them to describe the whole directory and not the search", counts)
	}

	gyms := s.do(http.MethodGet, "/admin/businesses?category_id="+itoa(s.categoryID("gimnasios")), root, nil)
	if rows, _ := gyms.body["businesses"].([]any); len(rows) != 0 {
		t.Errorf("another category = %v, want none (both businesses are yoga)", rows)
	}
	yoga := s.do(http.MethodGet, "/admin/businesses?category_id="+itoa(s.categoryID("yoga")), root, nil)
	if rows, _ := yoga.body["businesses"].([]any); len(rows) != 2 {
		t.Errorf("yoga = %v, want both", rows)
	}

	for _, query := range []string{"?category_id=abc", "?category_id=-1", "?q=" + strings.Repeat("a", 101)} {
		if got := s.do(http.MethodGet, "/admin/businesses"+query, root, nil); got.status != http.StatusUnprocessableEntity {
			t.Errorf("%s = %d, want 422", query[:12], got.status)
		}
	}
}

func TestAdminDetailShowsBranchesServicesAndHistory(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	_, root := s.user("root@example.com", "super_admin")
	owner := ownerWithBusiness(s, "owner@example.com")
	s.do(http.MethodPost, "/me/business/branches", owner, validBranch())
	s.do(http.MethodPost, "/me/business/services", owner, map[string]any{"name": "Yoga"})
	off := s.do(http.MethodPost, "/me/business/services", owner, map[string]any{"name": "Pilates"})
	s.do(http.MethodPut, "/me/business/services/"+serviceID(off)+"/active", owner, map[string]any{"active": false})
	s.do(http.MethodPost, "/me/business/submit", owner, payerBody("owner@example.com", "Olga Owner", nil))

	list := s.do(http.MethodGet, "/admin/businesses", root, nil)
	id := itoa(int64(list.body["businesses"].([]any)[0].(map[string]any)["id"].(float64)))
	adminAction(s, root, id, "approve")
	adminAction(s, root, id, "verify")

	got := s.do(http.MethodGet, "/admin/businesses/"+id, root, nil)
	business, _ := got.body["business"].(map[string]any)
	branches, _ := got.body["branches"].([]any)
	history, _ := got.body["history"].([]any)
	if got.status != http.StatusOK || business["owner_email"] != "owner@example.com" || business["mp_payer_email"] != "owner@example.com" || business["mp_payer_name"] != "Olga Owner" || business["whatsapp"] == "" {
		t.Fatalf("detail = %d %v", got.status, got.body)
	}
	if len(branches) != 1 || got.body["services_total"] != float64(2) || got.body["services_active"] != float64(1) {
		t.Errorf("branches %d, services %v/%v; want 1 branch and 1 of 2 services active", len(branches), got.body["services_active"], got.body["services_total"])
	}
	if len(history) != 2 || history[0].(map[string]any)["action"] != "business.verify" || history[0].(map[string]any)["actor_email"] != "root@example.com" {
		t.Errorf("history = %v, want the two admin actions, newest first, with who did them", history)
	}
}

func TestAdminDetailIsForSuperAdminsOnly(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	_, root := s.user("root@example.com", "super_admin")
	owner := ownerWithBusiness(s, "owner@example.com")
	id := itoa(int64(s.do(http.MethodGet, "/admin/businesses", root, nil).body["businesses"].([]any)[0].(map[string]any)["id"].(float64)))

	if got := s.do(http.MethodGet, "/admin/businesses/"+id, "", nil); got.status != http.StatusUnauthorized {
		t.Errorf("anonymous = %d, want 401", got.status)
	}
	if got := s.do(http.MethodGet, "/admin/businesses/"+id, owner, nil); got.status != http.StatusForbidden {
		t.Errorf("owner = %d, want 403", got.status)
	}
	if got := s.do(http.MethodGet, "/admin/businesses/99999", root, nil); got.status != http.StatusNotFound || got.errorCode() != "business_not_found" {
		t.Errorf("missing = %d %q, want 404 business_not_found", got.status, got.errorCode())
	}
	if got := s.do(http.MethodGet, "/admin/businesses/abc", root, nil); got.status != http.StatusNotFound {
		t.Errorf("non numeric id = %d, want 404", got.status)
	}
}
