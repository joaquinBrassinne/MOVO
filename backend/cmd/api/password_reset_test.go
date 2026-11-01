package main

import (
	"net/http"
	"sync"
	"testing"

	identitydomain "github.com/joaquinBrassinne/MOVO/internal/identity/domain"
)

const (
	oldPassword = "the old password 123"
	newPassword = "the brand new password 456"
)

// TestAssistedPasswordResetEndToEnd walks SDD 6.2 through the real router and database: the
// super admin issues a link for an owner, the owner sets a new password with it once, every
// old session ends, and the whole thing is in the audit log.
func TestAssistedPasswordResetEndToEnd(t *testing.T) {
	t.Parallel()

	h := newAPIHarness(t)
	adminToken := h.user("root@example.com", identitydomain.RoleSuperAdmin)

	status, session := h.do(http.MethodPost, "/auth/register", "", map[string]any{"email": "owner@example.com", "password": oldPassword})
	if status != http.StatusCreated {
		t.Fatalf("register = %d %v", status, session)
	}
	ownerToken, _ := session["access_token"].(string)
	oldRefresh, _ := session["refresh_token"].(string)

	issue := func() string {
		t.Helper()
		status, body := h.do(http.MethodPost, "/admin/password-resets", adminToken, map[string]any{"email": "Owner@Example.com"})
		token, _ := body["token"].(string)
		if status != http.StatusCreated || token == "" || body["expires_at"] == nil {
			t.Fatalf("issue = %d %v, want 201 with a token", status, body)
		}
		return token
	}

	// Only a super admin issues links, and only for owners.
	if status, _ := h.do(http.MethodPost, "/admin/password-resets", "", map[string]any{"email": "owner@example.com"}); status != http.StatusUnauthorized {
		t.Errorf("anonymous issue = %d, want 401", status)
	}
	if status, _ := h.do(http.MethodPost, "/admin/password-resets", ownerToken, map[string]any{"email": "owner@example.com"}); status != http.StatusForbidden {
		t.Errorf("owner issue = %d, want 403", status)
	}
	for _, email := range []string{"root@example.com", "nobody@example.com"} {
		if status, body := h.do(http.MethodPost, "/admin/password-resets", adminToken, map[string]any{"email": email}); status != http.StatusNotFound || errorCode(body) != "user_not_found" {
			t.Errorf("issue for %s = %d %q, want 404 user_not_found", email, status, errorCode(body))
		}
	}
	if status, body := h.do(http.MethodPost, "/admin/password-resets", adminToken, map[string]any{"email": "not an email"}); status != http.StatusUnprocessableEntity || errorCode(body) != "validation_failed" {
		t.Errorf("issue for a bad email = %d %q, want 422", status, errorCode(body))
	}

	// A newer link retires the previous one.
	first := issue()
	token := issue()
	if status, body := h.do(http.MethodPost, "/auth/reset-password", "", map[string]any{"token": first, "password": newPassword}); status != http.StatusBadRequest || errorCode(body) != "invalid_reset_token" {
		t.Errorf("a retired link = %d %q, want 400 invalid_reset_token", status, errorCode(body))
	}

	// A weak password is refused and leaves the link usable.
	if status, body := h.do(http.MethodPost, "/auth/reset-password", "", map[string]any{"token": token, "password": "short"}); status != http.StatusUnprocessableEntity || errorCode(body) != "validation_failed" {
		t.Errorf("weak password = %d %q, want 422", status, errorCode(body))
	}

	if status, body := h.do(http.MethodPost, "/auth/reset-password", "", map[string]any{"token": token, "password": newPassword}); status != http.StatusNoContent {
		t.Fatalf("reset = %d %v, want 204", status, body)
	}

	// It works once.
	if status, body := h.do(http.MethodPost, "/auth/reset-password", "", map[string]any{"token": token, "password": newPassword}); status != http.StatusBadRequest || errorCode(body) != "invalid_reset_token" {
		t.Errorf("reusing the link = %d %q, want 400 invalid_reset_token", status, errorCode(body))
	}

	// The new password signs in, the old one does not, and the old session is gone.
	if status, _ := h.do(http.MethodPost, "/auth/login", "", map[string]any{"email": "owner@example.com", "password": newPassword}); status != http.StatusOK {
		t.Errorf("login with the new password = %d, want 200", status)
	}
	if status, _ := h.do(http.MethodPost, "/auth/login", "", map[string]any{"email": "owner@example.com", "password": oldPassword}); status != http.StatusUnauthorized {
		t.Errorf("login with the old password = %d, want 401", status)
	}
	if status, _ := h.do(http.MethodPost, "/auth/refresh", "", map[string]any{"refresh_token": oldRefresh}); status != http.StatusUnauthorized {
		t.Errorf("refresh with a pre-reset token = %d, want 401", status)
	}

	var actions []string
	rows, err := h.pool.Query(t.Context(), `SELECT action FROM audit_log WHERE entity = 'user' ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			t.Fatal(err)
		}
		actions = append(actions, a)
	}
	want := []string{"user.password_reset_issue", "user.password_reset_issue", "user.password_reset"}
	if len(actions) != len(want) {
		t.Fatalf("audit actions = %v, want %v", actions, want)
	}
	for i := range want {
		if actions[i] != want[i] {
			t.Fatalf("audit actions = %v, want %v", actions, want)
		}
	}
}

func TestPasswordResetLinkExpiresAndRedeemsOnceUnderRace(t *testing.T) {
	t.Parallel()

	h := newAPIHarness(t)
	adminToken := h.user("root@example.com", identitydomain.RoleSuperAdmin)
	if status, body := h.do(http.MethodPost, "/auth/register", "", map[string]any{"email": "owner@example.com", "password": oldPassword}); status != http.StatusCreated {
		t.Fatalf("register = %d %v", status, body)
	}
	issue := func() string {
		t.Helper()
		_, body := h.do(http.MethodPost, "/admin/password-resets", adminToken, map[string]any{"email": "owner@example.com"})
		token, _ := body["token"].(string)
		if token == "" {
			t.Fatalf("no token in %v", body)
		}
		return token
	}

	// Past its 24 hours the link is dead.
	expired := issue()
	if _, err := h.pool.Exec(t.Context(), `UPDATE password_reset_tokens SET expires_at = now() - interval '1 second'`); err != nil {
		t.Fatal(err)
	}
	if status, body := h.do(http.MethodPost, "/auth/reset-password", "", map[string]any{"token": expired, "password": newPassword}); status != http.StatusBadRequest || errorCode(body) != "invalid_reset_token" {
		t.Errorf("expired link = %d %q, want 400 invalid_reset_token", status, errorCode(body))
	}

	// Two owners opening the same link at once: exactly one wins.
	token := issue()
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		statuses []int
	)
	for range 2 {
		wg.Go(func() {
			status, _ := h.do(http.MethodPost, "/auth/reset-password", "", map[string]any{"token": token, "password": newPassword})
			mu.Lock()
			statuses = append(statuses, status)
			mu.Unlock()
		})
	}
	wg.Wait()
	ok, bad := 0, 0
	for _, s := range statuses {
		switch s {
		case http.StatusNoContent:
			ok++
		case http.StatusBadRequest:
			bad++
		}
	}
	if ok != 1 || bad != 1 {
		t.Errorf("concurrent redeems = %v, want one 204 and one 400", statuses)
	}
}
