package commands_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/identity"
	"github.com/joaquinBrassinne/MOVO/internal/identity/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/identity/domain"
)

const (
	ownerEmail    = "owner@example.com"
	ownerPassword = "correct horse battery"
)

func registerOwner(t *testing.T, h *harness) commands.Session {
	t.Helper()

	session, err := h.register.Handle(t.Context(), commands.RegisterInput{Email: ownerEmail, Password: ownerPassword})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	return session
}

func TestRegisterCreatesABusinessAdminAndSignsIn(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	session, err := h.register.Handle(t.Context(), commands.RegisterInput{Email: "  Owner@Example.com", Password: ownerPassword})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	if session.User.Email != ownerEmail || session.User.Role != domain.RoleBusinessAdmin {
		t.Errorf("user = (%q, %q), want (%q, %q)", session.User.Email, session.User.Role, ownerEmail, domain.RoleBusinessAdmin)
	}
	if session.AccessToken != "access-1-business_admin" {
		t.Errorf("AccessToken = %q, want a token for the new business admin", session.AccessToken)
	}
	if want := epoch.Add(refreshTTL); !session.RefreshExpiresAt.Equal(want) {
		t.Errorf("RefreshExpiresAt = %v, want %v", session.RefreshExpiresAt, want)
	}
	if !h.store.holds(domain.HashRefreshToken(session.RefreshToken)) {
		t.Error("the store does not hold the digest of the issued refresh token")
	}
	if h.store.holds([]byte(session.RefreshToken)) {
		t.Error("the store holds the refresh token in plain text")
	}
}

func TestRegisterCountsEveryAttemptOfTheAddressAndSkipsUnknownOnes(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	// A failed attempt counts too: the budget is for tries, not for accounts.
	_, _ = h.register.Handle(t.Context(), commands.RegisterInput{Email: "nope", Password: ownerPassword, ClientIP: "203.0.113.7"})
	if _, err := h.register.Handle(t.Context(), commands.RegisterInput{Email: ownerEmail, Password: ownerPassword, ClientIP: "203.0.113.7"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	// Without an address there is nothing to count.
	if _, err := h.register.Handle(t.Context(), commands.RegisterInput{Email: "other@example.com", Password: ownerPassword}); err != nil {
		t.Fatalf("Register without address: %v", err)
	}

	if want := []string{"ip:203.0.113.7", "ip:203.0.113.7"}; !slices.Equal(h.limiter.attempts, want) {
		t.Errorf("attempts = %v, want %v", h.limiter.attempts, want)
	}
}

func TestRegisterIsRefusedWhileTheAddressIsRateLimitedAndStoresNothing(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.limiter.blocked["ip:203.0.113.7"] = 90 * time.Second

	_, err := h.register.Handle(t.Context(), commands.RegisterInput{Email: ownerEmail, Password: ownerPassword, ClientIP: "203.0.113.7"})

	var limited *commands.RateLimitedError
	if !errors.As(err, &limited) || limited.RetryAfter != 90*time.Second {
		t.Fatalf("Register error = %v, want a RateLimitedError with 90s", err)
	}
	// Nothing was stored: the same email registers fine from an address with budget left.
	if _, err := h.register.Handle(t.Context(), commands.RegisterInput{Email: ownerEmail, Password: ownerPassword, ClientIP: "198.51.100.9"}); err != nil {
		t.Errorf("Register after a refused attempt: %v, want the email to be free", err)
	}
}

func TestRegisterRejectsBadInputWithoutStoringAnything(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      commands.RegisterInput
		wantErr error
	}{
		{"bad email", commands.RegisterInput{Email: "nope", Password: ownerPassword}, domain.ErrEmailInvalid},
		{"short password", commands.RegisterInput{Email: ownerEmail, Password: "short"}, domain.ErrPasswordTooShort},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t)

			if _, err := h.register.Handle(t.Context(), tt.in); !errors.Is(err, tt.wantErr) {
				t.Fatalf("Register error = %v, want %v", err, tt.wantErr)
			}
			if len(h.users.byEmail) != 0 || len(h.store.tokens) != 0 {
				t.Error("Register stored data for rejected input")
			}
		})
	}
}

func TestRegisterRejectsADuplicateEmail(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	registerOwner(t, h)

	_, err := h.register.Handle(t.Context(), commands.RegisterInput{Email: "OWNER@example.com", Password: ownerPassword})
	if !errors.Is(err, identity.ErrEmailTaken) {
		t.Fatalf("Register error = %v, want %v", err, identity.ErrEmailTaken)
	}
}

func TestLoginSignsInWithValidCredentials(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	registerOwner(t, h)

	session, err := h.login.Handle(t.Context(), commands.LoginInput{Email: " OWNER@example.com ", Password: ownerPassword, ClientIP: "203.0.113.7"})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	if session.User.Email != ownerEmail || session.AccessToken == "" || session.RefreshToken == "" {
		t.Errorf("session = %+v, want the owner with both tokens", session)
	}
	if !slices.Equal(h.limiter.resets, []string{"email:" + ownerEmail}) {
		t.Errorf("resets = %v, want only the account key (never the address key)", h.limiter.resets)
	}
	if len(h.limiter.fails) != 0 {
		t.Errorf("fails = %v, want none", h.limiter.fails)
	}
}

func TestLoginRejectsWrongPasswordAndUnknownEmailIdentically(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		email string
		pass  string
	}{
		{"wrong password", ownerEmail, "not the password"},
		{"unknown email", "ghost@example.com", ownerPassword},
		{"malformed email", "nope", ownerPassword},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t)
			registerOwner(t, h)

			_, err := h.login.Handle(t.Context(), commands.LoginInput{Email: tt.email, Password: tt.pass, ClientIP: "203.0.113.7"})
			if !errors.Is(err, commands.ErrInvalidCredentials) {
				t.Fatalf("Login error = %v, want %v", err, commands.ErrInvalidCredentials)
			}
			if !slices.Contains(h.limiter.fails, "ip:203.0.113.7") {
				t.Errorf("fails = %v, want the address key recorded", h.limiter.fails)
			}
			if len(h.limiter.resets) != 0 {
				t.Errorf("resets = %v, want none after a failure", h.limiter.resets)
			}
		})
	}
}

func TestLoginFailureCountsAgainstTheAccount(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	registerOwner(t, h)

	_, _ = h.login.Handle(t.Context(), commands.LoginInput{Email: ownerEmail, Password: "wrong password", ClientIP: "203.0.113.7"})

	if want := []string{"ip:203.0.113.7", "email:" + ownerEmail}; !slices.Equal(h.limiter.fails, want) {
		t.Errorf("fails = %v, want %v", h.limiter.fails, want)
	}
}

func TestLoginIsRefusedWhileRateLimitedAndDoesNotCheckTheAccount(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"account key": "email:" + ownerEmail,
		"address key": "ip:203.0.113.7",
	}
	for name, key := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t)
			registerOwner(t, h)
			h.limiter.blocked[key] = 90 * time.Second

			// The password is right: a blocked key must still be refused.
			_, err := h.login.Handle(t.Context(), commands.LoginInput{Email: ownerEmail, Password: ownerPassword, ClientIP: "203.0.113.7"})

			var limited *commands.RateLimitedError
			if !errors.As(err, &limited) || limited.RetryAfter != 90*time.Second {
				t.Fatalf("Login error = %v, want a RateLimitedError with 90s", err)
			}
			if h.users.lookups != 0 {
				t.Errorf("user lookups = %d, want 0 while limited", h.users.lookups)
			}
			if len(h.limiter.fails) != 0 {
				t.Errorf("fails = %v, want a refused attempt not to count again", h.limiter.fails)
			}
		})
	}
}

func TestRefreshRotatesTheTokenAndKillsTheOldOne(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	first := registerOwner(t, h)

	second, err := h.refresh.Handle(t.Context(), first.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	if second.RefreshToken == first.RefreshToken {
		t.Error("Refresh returned the same refresh token, want a rotated one")
	}
	if second.User.Email != ownerEmail || second.AccessToken == "" {
		t.Errorf("session = %+v, want the owner with a new access token", second)
	}

	if _, err := h.refresh.Handle(t.Context(), first.RefreshToken); !errors.Is(err, commands.ErrSessionInvalid) {
		t.Errorf("old token error = %v, want %v", err, commands.ErrSessionInvalid)
	}
}

func TestRefreshReuseRevokesEveryTokenOfTheUser(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	first := registerOwner(t, h)

	second, err := h.refresh.Handle(t.Context(), first.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	// The stolen, already rotated token comes back: the whole family must die.
	if _, err := h.refresh.Handle(t.Context(), first.RefreshToken); !errors.Is(err, commands.ErrSessionInvalid) {
		t.Fatalf("reused token error = %v, want %v", err, commands.ErrSessionInvalid)
	}
	if _, err := h.refresh.Handle(t.Context(), second.RefreshToken); !errors.Is(err, commands.ErrSessionInvalid) {
		t.Errorf("legitimate successor error = %v, want it revoked with the family", err)
	}
}

func TestRefreshRejectsUnusableTokens(t *testing.T) {
	t.Parallel()

	for name, token := range map[string]string{"empty": "", "unknown": "never-issued"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t)

			if _, err := h.refresh.Handle(t.Context(), token); !errors.Is(err, commands.ErrSessionInvalid) {
				t.Errorf("Refresh(%q) error = %v, want %v", token, err, commands.ErrSessionInvalid)
			}
		})
	}
}

func TestLogoutEndsTheSessionAndIsIdempotent(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	session := registerOwner(t, h)

	if err := h.logout.Handle(t.Context(), session.RefreshToken); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if _, err := h.refresh.Handle(t.Context(), session.RefreshToken); !errors.Is(err, commands.ErrSessionInvalid) {
		t.Errorf("Refresh after Logout error = %v, want %v", err, commands.ErrSessionInvalid)
	}

	for name, token := range map[string]string{"again": session.RefreshToken, "unknown": "never-issued", "empty": ""} {
		if err := h.logout.Handle(t.Context(), token); err != nil {
			t.Errorf("Logout(%s) error = %v, want nil", name, err)
		}
	}
}
