package businesshttp_test

import (
	"net/http"
	"testing"
)

func serviceID(r reply) string { return itoa(int64(r.body["id"].(float64))) }

func TestOwnerManagesServicesAndTheirSchedule(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	owner := ownerWithBusiness(s, "owner@example.com")
	branch := s.do(http.MethodPost, "/me/business/branches", owner, validBranch())

	created := s.do(http.MethodPost, "/me/business/services", owner, map[string]any{
		"name": "Yoga Flow", "description": "Para todos", "branch_id": branch.body["id"],
	})
	if created.status != http.StatusCreated || created.body["name"] != "Yoga Flow" || created.body["active"] != true || created.body["branch_id"] != branch.body["id"] {
		t.Fatalf("create service = %d %v", created.status, created.body)
	}
	whole := s.do(http.MethodPost, "/me/business/services", owner, map[string]any{"name": "Cancha"})
	if whole.status != http.StatusCreated || whole.body["branch_id"] != nil {
		t.Fatalf("service without branch = %d %v, want branch_id null", whole.status, whole.body)
	}
	path := "/me/business/services/" + serviceID(created)

	schedule := map[string]any{"slots": []map[string]any{
		{"weekday": 1, "start": "09:00", "end": "10:00"},
		{"weekday": 3, "start": "18:00", "end": "19:00"},
	}}
	if got := s.do(http.MethodPut, path+"/schedule", owner, schedule); got.status != http.StatusOK {
		t.Fatalf("set schedule = %d %v", got.status, got.body)
	}
	if got := s.do(http.MethodPut, path, owner, map[string]any{"name": "Yoga Renamed", "branch_id": nil}); got.status != http.StatusOK || got.body["name"] != "Yoga Renamed" || got.body["branch_id"] != nil {
		t.Fatalf("edit = %d %v", got.status, got.body)
	}
	if got := s.do(http.MethodPut, path+"/active", owner, map[string]any{"active": false}); got.status != http.StatusOK || got.body["active"] != false {
		t.Fatalf("deactivate = %d %v", got.status, got.body)
	}

	list := s.do(http.MethodGet, "/me/business/services", owner, nil)
	services, _ := list.body["services"].([]any)
	if list.status != http.StatusOK || len(services) != 2 {
		t.Fatalf("list = %d %v, want both services, the inactive one included", list.status, list.body)
	}
	first, _ := services[0].(map[string]any)
	slots, _ := first["schedule"].([]any)
	if first["name"] != "Yoga Renamed" || first["active"] != false || len(slots) != 2 {
		t.Errorf("first service = %v", first)
	}
	if second, _ := services[1].(map[string]any); second["schedule"] == nil {
		t.Error("a service without schedule must list [] and not null")
	}

	if got := s.do(http.MethodDelete, path, owner, nil); got.status != http.StatusNoContent {
		t.Fatalf("delete = %d", got.status)
	}
	if got := s.do(http.MethodDelete, path, owner, nil); got.status != http.StatusNotFound || got.errorCode() != "service_not_found" {
		t.Errorf("second delete = %d %q, want 404 service_not_found", got.status, got.errorCode())
	}
}

func TestServiceLimitCountsActiveServicesOnly(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{services: 2})
	owner := ownerWithBusiness(s, "owner@example.com")

	add := func(name string) reply {
		return s.do(http.MethodPost, "/me/business/services", owner, map[string]any{"name": name})
	}
	first, second := add("One"), add("Two")
	if first.status != http.StatusCreated || second.status != http.StatusCreated {
		t.Fatalf("services within the limit = %d, %d", first.status, second.status)
	}

	got := add("Three")
	if got.status != http.StatusConflict || got.errorCode() != "plan_limit_reached" {
		t.Fatalf("third service = %d %q, want 409 plan_limit_reached", got.status, got.errorCode())
	}

	firstPath := "/me/business/services/" + serviceID(first)
	if got := s.do(http.MethodPut, firstPath+"/active", owner, map[string]any{"active": false}); got.status != http.StatusOK {
		t.Fatalf("deactivate = %d", got.status)
	}
	if got := add("Three"); got.status != http.StatusCreated {
		t.Fatalf("a deactivated service must free its place, got %d", got.status)
	}
	if got := s.do(http.MethodPut, firstPath+"/active", owner, map[string]any{"active": true}); got.status != http.StatusConflict || got.errorCode() != "plan_limit_reached" {
		t.Errorf("activation over the limit = %d %q, want 409 plan_limit_reached", got.status, got.errorCode())
	}
}

func TestServiceValidation(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	owner := ownerWithBusiness(s, "owner@example.com")
	other := ownerWithBusiness(s, "other@example.com")
	foreignBranch := s.do(http.MethodPost, "/me/business/branches", other, validBranch())

	tests := []struct {
		name      string
		body      any
		wantCode  int
		wantError string
		wantField string
	}{
		{"empty name", map[string]any{"name": " "}, http.StatusUnprocessableEntity, "validation_failed", "name"},
		{"branch of another business", map[string]any{"name": "x", "branch_id": foreignBranch.body["id"]}, http.StatusNotFound, "branch_not_found", ""},
		{"branch that does not exist", map[string]any{"name": "x", "branch_id": 99999}, http.StatusNotFound, "branch_not_found", ""},
		{"negative branch", map[string]any{"name": "x", "branch_id": -1}, http.StatusUnprocessableEntity, "validation_failed", "branch_id"},
		{"unknown field", map[string]any{"name": "x", "business_id": 3}, http.StatusBadRequest, "invalid_request", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.do(http.MethodPost, "/me/business/services", owner, tt.body)
			if got.status != tt.wantCode || got.errorCode() != tt.wantError || got.errorField() != tt.wantField {
				t.Fatalf("got %d %q field %q, want %d %q field %q (%v)", got.status, got.errorCode(), got.errorField(), tt.wantCode, tt.wantError, tt.wantField, got.body)
			}
		})
	}

	svc := s.do(http.MethodPost, "/me/business/services", owner, map[string]any{"name": "Yoga"})
	overlapping := map[string]any{"slots": []map[string]any{
		{"weekday": 1, "start": "09:00", "end": "11:00"},
		{"weekday": 1, "start": "10:00", "end": "12:00"},
	}}
	if got := s.do(http.MethodPut, "/me/business/services/"+serviceID(svc)+"/schedule", owner, overlapping); got.status != http.StatusUnprocessableEntity || got.errorField() != "slots" {
		t.Errorf("overlapping schedule = %d field %q, want 422 slots", got.status, got.errorField())
	}
}

func TestServicesOfAnotherBusinessAreUnreachable(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	alice := ownerWithBusiness(s, "alice@example.com")
	bob := ownerWithBusiness(s, "bob@example.com")

	svc := s.do(http.MethodPost, "/me/business/services", alice, map[string]any{"name": "Alice Yoga"})
	path := "/me/business/services/" + serviceID(svc)

	for name, got := range map[string]reply{
		"edit":     s.do(http.MethodPut, path, bob, map[string]any{"name": "Hijacked"}),
		"delete":   s.do(http.MethodDelete, path, bob, nil),
		"active":   s.do(http.MethodPut, path+"/active", bob, map[string]any{"active": false}),
		"schedule": s.do(http.MethodPut, path+"/schedule", bob, map[string]any{"slots": []any{}}),
	} {
		if got.status != http.StatusNotFound || got.errorCode() != "service_not_found" {
			t.Errorf("bob %s Alice's service = %d %q, want 404 service_not_found", name, got.status, got.errorCode())
		}
	}

	bobs := s.do(http.MethodGet, "/me/business/services", bob, nil)
	if list, _ := bobs.body["services"].([]any); len(list) != 0 {
		t.Errorf("bob's services = %v, want none", list)
	}
	alices := s.do(http.MethodGet, "/me/business/services", alice, nil)
	if list, _ := alices.body["services"].([]any); len(list) != 1 {
		t.Fatalf("alice's services = %v", list)
	} else if first, _ := list[0].(map[string]any); first["name"] != "Alice Yoga" || first["active"] != true {
		t.Errorf("alice's service changed: %v", first)
	}
}

func TestServicePriceRoundTripsAndIsValidated(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	owner := ownerWithBusiness(s, "owner@example.com")

	created := s.do(http.MethodPost, "/me/business/services", owner, map[string]any{"name": "Yoga", "price": 5500})
	if created.status != http.StatusCreated || created.body["price"] != float64(5500) {
		t.Fatalf("priced service = %d %v", created.status, created.body)
	}
	free := s.do(http.MethodPost, "/me/business/services", owner, map[string]any{"name": "Libre"})
	if free.status != http.StatusCreated || free.body["price"] != nil {
		t.Fatalf("service without price = %d %v, want price null", free.status, free.body)
	}

	list := s.do(http.MethodGet, "/me/business/services", owner, nil)
	first, _ := list.body["services"].([]any)[0].(map[string]any)
	if first["price"] != float64(5500) {
		t.Errorf("listed price = %v", first["price"])
	}

	for _, price := range []int{-5, 100_000_001} {
		got := s.do(http.MethodPost, "/me/business/services", owner, map[string]any{"name": "x", "price": price})
		if got.status != http.StatusUnprocessableEntity || got.errorField() != "price" {
			t.Errorf("price %d = %d field %q, want 422 price", price, got.status, got.errorField())
		}
	}
}
