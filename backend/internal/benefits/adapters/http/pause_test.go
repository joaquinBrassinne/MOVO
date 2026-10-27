package benefitshttp_test

import (
	"net/http"
	"strings"
	"testing"
)

// benefitID extracts the id from the public coupon path "/benefits/{id}/codes".
func benefitID(couponPath string) string {
	return strings.TrimSuffix(strings.TrimPrefix(couponPath, "/benefits/"), "/codes")
}

func TestSuperAdminPausesABenefitAndTheOwnerCannotLiftIt(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	path := s.visibleBenefit("owner@example.com", 5)
	owner := "token-owner@example.com"
	adminID, admin := s.user("admin@example.com", "super_admin")
	pause := "/admin/benefits/" + benefitID(path) + "/paused"

	paused := s.do(http.MethodPut, pause, admin, map[string]any{"paused": true})
	if paused.status != http.StatusOK || paused.body["admin_paused"] != true || paused.body["active"] != true {
		t.Fatalf("pause = %d %v", paused.status, paused.body)
	}

	// Customers can no longer claim it.
	if got := s.do(http.MethodPost, path, "", customer("Ana", "5491112345678")); got.status != http.StatusNotFound || got.errorCode() != "benefit_not_found" {
		t.Errorf("claim while paused = %d %q, want 404 benefit_not_found", got.status, got.errorCode())
	}

	// The owner sees the pause and cannot lift it, however they toggle their own switch.
	ownerPath := "/me/business/benefits/" + benefitID(path)
	s.do(http.MethodPut, ownerPath+"/active", owner, map[string]any{"active": false})
	if back := s.do(http.MethodPut, ownerPath+"/active", owner, map[string]any{"active": true}); back.body["admin_paused"] != true {
		t.Errorf("after the owner toggles = %v, want admin_paused still true", back.body)
	}
	edit := s.do(http.MethodPut, ownerPath, owner, map[string]any{"type": "percent", "percent": 20})
	if edit.status != http.StatusOK || edit.body["admin_paused"] != true {
		t.Errorf("after the owner edits = %d %v, want admin_paused still true", edit.status, edit.body)
	}
	if got := s.do(http.MethodPost, path, "", customer("Ana", "5491112345678")); got.status != http.StatusNotFound {
		t.Errorf("claim after the owner's toggles = %d, want it still hidden", got.status)
	}
	// The owner cannot reach the admin route, nor send the flag through their own.
	if got := s.do(http.MethodPut, pause, owner, map[string]any{"paused": false}); got.status != http.StatusForbidden {
		t.Errorf("owner on the admin route = %d, want 403", got.status)
	}
	if got := s.do(http.MethodPut, ownerPath, owner, map[string]any{"type": "percent", "percent": 20, "admin_paused": false}); got.status != http.StatusBadRequest {
		t.Errorf("owner sending admin_paused = %d, want 400 (unknown field)", got.status)
	}

	// Resuming makes it claimable again.
	if resumed := s.do(http.MethodPut, pause, admin, map[string]any{"paused": false}); resumed.status != http.StatusOK || resumed.body["admin_paused"] != false {
		t.Fatalf("resume = %d %v", resumed.status, resumed.body)
	}
	if got := s.do(http.MethodPost, path, "", customer("Ana", "5491112345678")); got.status != http.StatusCreated {
		t.Errorf("claim after resume = %d %v, want 201", got.status, got.body)
	}

	// Every change is on record, the repeat is not.
	s.do(http.MethodPut, pause, admin, map[string]any{"paused": false})
	var actions []string
	rows, err := s.pool.Query(t.Context(), `SELECT action FROM audit_log WHERE entity = 'benefit' AND actor_id = $1 ORDER BY id`, adminID)
	if err != nil {
		t.Fatalf("read audit: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			t.Fatalf("scan: %v", err)
		}
		actions = append(actions, a)
	}
	if strings.Join(actions, ",") != "benefit.pause,benefit.resume" {
		t.Errorf("audit actions = %v, want pause and resume once each", actions)
	}
}

func TestPauseNeedsASuperAdminAndAnExistingBenefit(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	s.visibleBenefit("owner@example.com", 5)
	_, admin := s.user("admin@example.com", "super_admin")

	if got := s.do(http.MethodPut, "/admin/benefits/1/paused", "", map[string]any{"paused": true}); got.status != http.StatusUnauthorized {
		t.Errorf("no token = %d, want 401", got.status)
	}
	if got := s.do(http.MethodPut, "/admin/benefits/999999/paused", admin, map[string]any{"paused": true}); got.status != http.StatusNotFound || got.errorCode() != "benefit_not_found" {
		t.Errorf("unknown benefit = %d %q, want 404 benefit_not_found", got.status, got.errorCode())
	}
	if got := s.do(http.MethodPut, "/admin/benefits/abc/paused", admin, map[string]any{"paused": true}); got.status != http.StatusNotFound {
		t.Errorf("malformed id = %d, want 404", got.status)
	}
}

func TestOwnerCreatesBenefitsOfEveryTypeWithTheirLabels(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	owner := s.owner("owner@example.com", "active")

	for _, tc := range []struct {
		name  string
		body  map[string]any
		label string
	}{
		{"any percent", map[string]any{"type": "percent", "percent": 37}, "-37%"},
		{"fixed amount", map[string]any{"type": "amount", "amount_ars": 2000, "conditions": "Mínimo $10.000"}, "-$2.000"},
		{"custom", map[string]any{"type": "custom", "title": "Una clase gratis"}, "Beneficio"},
	} {
		got := s.do(http.MethodPost, "/me/business/benefits", owner, tc.body)
		if got.status != http.StatusCreated || got.body["short_label"] != tc.label {
			t.Errorf("%s = %d %v, want label %q", tc.name, got.status, got.body, tc.label)
		}
	}
	amount := s.do(http.MethodPost, "/me/business/benefits", owner, map[string]any{"type": "amount", "amount_ars": 500})
	if amount.body["amount_ars"] != float64(500) || amount.body["percent"] != nil || amount.body["title"] != nil || amount.body["conditions"] != nil {
		t.Errorf("amount body = %v, want only the amount set", amount.body)
	}
}
