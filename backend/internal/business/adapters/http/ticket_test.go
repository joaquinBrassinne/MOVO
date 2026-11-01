package businesshttp_test

import (
	"net/http"
	"testing"
)

func TestAProOwnerSetsKeepsAndClearsTheAverageTicket(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{advancedMetrics: true})
	_, root := s.user("root@example.com", "super_admin")
	owner := ownerWithBusiness(s, "owner@example.com")

	edit := validBusiness(s)
	edit["average_ticket_ars"] = 15000
	got := s.do(http.MethodPut, "/me/business", owner, edit)
	if got.status != http.StatusOK || got.body["average_ticket_ars"] != float64(15000) {
		t.Fatalf("set = %d %v, want 200 with the ticket 15000", got.status, got.body)
	}

	// A body without the field keeps what is stored.
	delete(edit, "average_ticket_ars")
	if got := s.do(http.MethodPut, "/me/business", owner, edit); got.status != http.StatusOK || got.body["average_ticket_ars"] != float64(15000) {
		t.Errorf("absent = %d %v, want the ticket kept", got.status, got.body)
	}
	if got := s.do(http.MethodGet, "/me/business", owner, nil); got.status != http.StatusOK ||
		got.body["business"].(map[string]any)["average_ticket_ars"] != float64(15000) {
		t.Errorf("read back = %d %v, want 15000", got.status, got.body)
	}

	// An explicit null removes it.
	edit["average_ticket_ars"] = nil
	if got := s.do(http.MethodPut, "/me/business", owner, edit); got.status != http.StatusOK || got.body["average_ticket_ars"] != nil {
		t.Errorf("null = %d %v, want the ticket removed", got.status, got.body)
	}

	// The change is in the admin history, with its old and new value.
	list := s.do(http.MethodGet, "/admin/businesses", root, nil)
	id := itoa(int64(list.body["businesses"].([]any)[0].(map[string]any)["id"].(float64)))
	history, _ := s.do(http.MethodGet, "/admin/businesses/"+id, root, nil).body["history"].([]any)
	if len(history) != 2 {
		t.Fatalf("history = %v, want one entry for setting it and one for clearing it", history)
	}
}

func TestTheAverageTicketMustBeAWholeNumberAboveZero(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{advancedMetrics: true})
	owner := ownerWithBusiness(s, "owner@example.com")

	for name, value := range map[string]any{
		"zero": 0, "negative": -5, "decimal": 12.5, "text": "mucho", "too large": 2147483648, "boolean": true,
	} {
		edit := validBusiness(s)
		edit["average_ticket_ars"] = value
		got := s.do(http.MethodPut, "/me/business", owner, edit)
		details, _ := got.body["error"].(map[string]any)["details"].(map[string]any)
		if got.status != http.StatusUnprocessableEntity || details["field"] != "average_ticket_ars" {
			t.Errorf("%s: %d %v, want 422 on average_ticket_ars", name, got.status, got.body)
		}
	}
}

func TestAStarterOwnerCannotSetTheAverageTicketButCanStillEditTheProfile(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	owner := ownerWithBusiness(s, "owner@example.com")

	edit := validBusiness(s)
	edit["average_ticket_ars"] = 15000
	got := s.do(http.MethodPut, "/me/business", owner, edit)
	if got.status != http.StatusForbidden || got.body["error"].(map[string]any)["code"] != "plan_upgrade_required" {
		t.Fatalf("set = %d %v, want 403 plan_upgrade_required", got.status, got.body)
	}

	delete(edit, "average_ticket_ars")
	edit["name"] = "Yoga Renamed"
	if got := s.do(http.MethodPut, "/me/business", owner, edit); got.status != http.StatusOK || got.body["name"] != "Yoga Renamed" || got.body["average_ticket_ars"] != nil {
		t.Errorf("edit without the ticket = %d %v, want 200 and no ticket", got.status, got.body)
	}
}

func TestACreateBodyWithAnAverageTicketIsRefusedNotIgnored(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{advancedMetrics: true})
	_, token := s.user("owner@example.com", "business_admin")

	body := validBusiness(s)
	body["average_ticket_ars"] = 15000
	if got := s.do(http.MethodPost, "/me/business", token, body); got.status == http.StatusCreated {
		t.Errorf("create with a ticket = %d %v, want it refused instead of silently dropping the field", got.status, got.body)
	}
}
