package businesshttp_test

import (
	"net/http"
	"strings"
	"testing"
)

func validBranch() map[string]any {
	return map[string]any{
		"name": "Centro", "address": "Honduras 4820", "city": "Buenos Aires",
		"latitude": -34.5875, "longitude": -58.4306, "booking_url": "https://example.com/reservar",
	}
}

// ownerWithBusiness creates an owner with a draft business and returns the token.
func ownerWithBusiness(s *server, email string) string {
	s.t.Helper()

	_, token := s.user(email, "business_admin")
	if got := s.do(http.MethodPost, "/me/business", token, validBusiness(s)); got.status != http.StatusCreated {
		s.t.Fatalf("create business for %s = %d %v", email, got.status, got.body)
	}
	return token
}

func branchID(r reply) string { return itoa(int64(r.body["id"].(float64))) }

func TestOwnerManagesBranchesAndTheirHours(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	owner := ownerWithBusiness(s, "owner@example.com")

	created := s.do(http.MethodPost, "/me/business/branches", owner, validBranch())
	if created.status != http.StatusCreated || created.body["name"] != "Centro" || created.body["active"] != true || created.body["latitude"] != -34.5875 {
		t.Fatalf("create branch = %d %v", created.status, created.body)
	}
	path := "/me/business/branches/" + branchID(created)

	edit := validBranch()
	edit["name"] = "Palermo"
	edit["booking_url"] = ""
	if got := s.do(http.MethodPut, path, owner, edit); got.status != http.StatusOK || got.body["name"] != "Palermo" || got.body["booking_url"] != "" {
		t.Fatalf("edit branch = %d %v", got.status, got.body)
	}

	hours := map[string]any{"slots": []map[string]any{
		{"weekday": 1, "start": "09:00", "end": "13:00"},
		{"weekday": 1, "start": "16:00", "end": "20:00"},
	}}
	if got := s.do(http.MethodPut, path+"/hours", owner, hours); got.status != http.StatusOK {
		t.Fatalf("set hours = %d %v", got.status, got.body)
	}

	mine := s.do(http.MethodGet, "/me/business", owner, nil)
	branches, _ := mine.body["branches"].([]any)
	if len(branches) != 1 {
		t.Fatalf("branches = %v", mine.body["branches"])
	}
	branch, _ := branches[0].(map[string]any)
	slots, _ := branch["hours"].([]any)
	if branch["name"] != "Palermo" || len(slots) != 2 {
		t.Errorf("branch = %v, want the edit and two slots", branch)
	}
	if first, _ := slots[0].(map[string]any); first["start"] != "09:00" || first["end"] != "13:00" {
		t.Errorf("first slot = %v", first)
	}

	if got := s.do(http.MethodDelete, path, owner, nil); got.status != http.StatusNoContent {
		t.Fatalf("delete = %d", got.status)
	}
	if got := s.do(http.MethodDelete, path, owner, nil); got.status != http.StatusNotFound || got.errorCode() != "branch_not_found" {
		t.Errorf("second delete = %d %q, want 404 branch_not_found", got.status, got.errorCode())
	}
}

func TestBranchValidation(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	owner := ownerWithBusiness(s, "owner@example.com")

	with := func(key string, value any) map[string]any {
		body := validBranch()
		body[key] = value
		return body
	}
	tests := []struct {
		name      string
		body      any
		wantCode  int
		wantField string
	}{
		{"empty name", with("name", ""), http.StatusUnprocessableEntity, "name"},
		{"empty address", with("address", " "), http.StatusUnprocessableEntity, "address"},
		{"empty city", with("city", ""), http.StatusUnprocessableEntity, "city"},
		{"pin never placed", map[string]any{"name": "x", "address": "x", "city": "x", "latitude": 0, "longitude": 0}, http.StatusUnprocessableEntity, "location"},
		{"latitude out of range", with("latitude", 91), http.StatusUnprocessableEntity, "location"},
		{"unsafe booking url", with("booking_url", "javascript:alert(1)"), http.StatusUnprocessableEntity, "booking_url"},
		{"unknown field", with("business_id", 7), http.StatusBadRequest, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.do(http.MethodPost, "/me/business/branches", owner, tt.body)
			if got.status != tt.wantCode || got.errorField() != tt.wantField {
				t.Fatalf("got %d field %q, want %d field %q (%v)", got.status, got.errorField(), tt.wantCode, tt.wantField, got.body)
			}
		})
	}
}

func TestHoursValidation(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	owner := ownerWithBusiness(s, "owner@example.com")
	path := "/me/business/branches/" + branchID(s.do(http.MethodPost, "/me/business/branches", owner, validBranch())) + "/hours"

	slot := func(weekday int, start, end string) map[string]any {
		return map[string]any{"weekday": weekday, "start": start, "end": end}
	}
	for name, slots := range map[string][]map[string]any{
		"overlap":          {slot(1, "09:00", "13:00"), slot(1, "12:00", "20:00")},
		"end before start": {slot(1, "13:00", "09:00")},
		"weekday 7":        {slot(7, "09:00", "13:00")},
		"not a time":       {slot(1, "9am", "13:00")},
		"25 o'clock":       {slot(1, "25:00", "26:00")},
	} {
		got := s.do(http.MethodPut, path, owner, map[string]any{"slots": slots})
		if got.status != http.StatusUnprocessableEntity || got.errorField() != "slots" {
			t.Errorf("%s = %d field %q, want 422 slots (%v)", name, got.status, got.errorField(), got.body)
		}
	}

	if got := s.do(http.MethodPut, path, owner, map[string]any{"slots": []any{}}); got.status != http.StatusOK {
		t.Errorf("closing the branch all week = %d, want 200", got.status)
	}
}

func TestBranchLimitOfThePlan(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{branches: 2})
	owner := ownerWithBusiness(s, "owner@example.com")
	other := ownerWithBusiness(s, "other@example.com")

	for range 2 {
		if got := s.do(http.MethodPost, "/me/business/branches", owner, validBranch()); got.status != http.StatusCreated {
			t.Fatalf("branch within the limit = %d %v", got.status, got.body)
		}
	}
	got := s.do(http.MethodPost, "/me/business/branches", owner, validBranch())
	if got.status != http.StatusConflict || got.errorCode() != "plan_limit_reached" {
		t.Fatalf("third branch = %d %q, want 409 plan_limit_reached", got.status, got.errorCode())
	}
	if e, _ := got.body["error"].(map[string]any); e["details"].(map[string]any)["resource"] != "branches" {
		t.Errorf("details = %v, want resource=branches", e["details"])
	}

	if got := s.do(http.MethodPost, "/me/business/branches", other, validBranch()); got.status != http.StatusCreated {
		t.Errorf("another business must have its own count, got %d", got.status)
	}
}

func TestBranchesOfAnotherBusinessAreUnreachable(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	alice := ownerWithBusiness(s, "alice@example.com")
	bob := ownerWithBusiness(s, "bob@example.com")

	aliceBranch := s.do(http.MethodPost, "/me/business/branches", alice, validBranch())
	path := "/me/business/branches/" + branchID(aliceBranch)

	hijack := validBranch()
	hijack["name"] = "Hijacked"
	for name, got := range map[string]reply{
		"edit":   s.do(http.MethodPut, path, bob, hijack),
		"delete": s.do(http.MethodDelete, path, bob, nil),
		"hours":  s.do(http.MethodPut, path+"/hours", bob, map[string]any{"slots": []any{}}),
	} {
		if got.status != http.StatusNotFound || got.errorCode() != "branch_not_found" {
			t.Errorf("bob %s Alice's branch = %d %q, want 404 branch_not_found", name, got.status, got.errorCode())
		}
	}

	alices := s.do(http.MethodGet, "/me/business", alice, nil)
	branches, _ := alices.body["branches"].([]any)
	if first, _ := branches[0].(map[string]any); len(branches) != 1 || first["name"] != "Centro" {
		t.Errorf("alice's branches = %v, must be untouched", branches)
	}

	if got := s.do(http.MethodPut, "/me/business/branches/not-a-number", alice, validBranch()); got.status != http.StatusNotFound {
		t.Errorf("non numeric id = %d, want 404", got.status)
	}
}

func TestBranchReferenceRoundTripsAndIsBounded(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	owner := ownerWithBusiness(s, "owner@example.com")

	body := validBranch()
	body["reference"] = "PB, timbre 4B"
	created := s.do(http.MethodPost, "/me/business/branches", owner, body)
	if created.status != http.StatusCreated || created.body["reference"] != "PB, timbre 4B" {
		t.Fatalf("branch with reference = %d %v", created.status, created.body)
	}

	body["reference"] = strings.Repeat("a", 101)
	if got := s.do(http.MethodPost, "/me/business/branches", owner, body); got.status != http.StatusUnprocessableEntity || got.errorField() != "reference" {
		t.Errorf("long reference = %d field %q, want 422 reference", got.status, got.errorField())
	}
}

func TestBranchVisibilityToggleRespectsThePlan(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{branches: 1})
	owner := ownerWithBusiness(s, "owner@example.com")
	other := ownerWithBusiness(s, "other@example.com")

	first := s.do(http.MethodPost, "/me/business/branches", owner, validBranch())
	path := "/me/business/branches/" + branchID(first) + "/active"

	if got := s.do(http.MethodPut, path, owner, map[string]any{"active": false}); got.status != http.StatusOK || got.body["active"] != false {
		t.Fatalf("hide = %d %v", got.status, got.body)
	}
	if got := s.do(http.MethodPost, "/me/business/branches", owner, validBranch()); got.status != http.StatusCreated {
		t.Fatalf("a hidden branch must free its place, got %d", got.status)
	}
	if got := s.do(http.MethodPut, path, owner, map[string]any{"active": true}); got.status != http.StatusConflict || got.errorCode() != "plan_limit_reached" {
		t.Errorf("show over the limit = %d %q, want 409 plan_limit_reached", got.status, got.errorCode())
	}
	if got := s.do(http.MethodPut, path, other, map[string]any{"active": true}); got.status != http.StatusNotFound {
		t.Errorf("foreign branch = %d, want 404", got.status)
	}
	if got := s.do(http.MethodPut, path, owner, map[string]any{"active": true, "extra": 1}); got.status != http.StatusBadRequest {
		t.Errorf("unknown field = %d, want 400", got.status)
	}
}
