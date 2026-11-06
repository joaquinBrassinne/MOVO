package businesshttp_test

import (
	"net/http"
	"testing"
)

func TestAnOwnersProfileChangeShowsUpInTheAdminHistory(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{})
	_, root := s.user("root@example.com", "super_admin")
	owner := ownerWithBusiness(s, "owner@example.com")

	edit := validBusiness(s)
	edit["name"] = "Yoga Renamed"
	if got := s.do(http.MethodPut, "/me/business", owner, edit); got.status != http.StatusOK {
		t.Fatalf("edit = %d %v", got.status, got.body)
	}
	// Sending the same profile again changes nothing and adds nothing to the history.
	if got := s.do(http.MethodPut, "/me/business", owner, edit); got.status != http.StatusOK {
		t.Fatalf("repeat = %d %v", got.status, got.body)
	}

	list := s.do(http.MethodGet, "/admin/businesses", root, nil)
	id := itoa(int64(list.body["businesses"].([]any)[0].(map[string]any)["id"].(float64)))
	detail := s.do(http.MethodGet, "/admin/businesses/"+id, root, nil)
	history, _ := detail.body["history"].([]any)
	if len(history) != 1 {
		t.Fatalf("history = %v, want one entry for the one change", history)
	}
	entry, _ := history[0].(map[string]any)
	changes, _ := entry["metadata"].(map[string]any)["changes"].(map[string]any)
	name, _ := changes["name"].(map[string]any)
	if entry["action"] != "business.profile_update" || entry["actor_email"] != "owner@example.com" || name["to"] != "Yoga Renamed" || len(changes) != 1 {
		t.Errorf("entry = %v, want the owner's change of the name", entry)
	}
}
