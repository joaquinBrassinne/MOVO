package commands_test

import (
	"context"
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/identity"
	"github.com/joaquinBrassinne/MOVO/internal/identity/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/identity/domain"
)

const (
	adminEmail    = "root@example.com"
	adminPassword = "a long enough password"
)

func TestEnsureSuperAdminCreatesTheAccountOnce(t *testing.T) {
	t.Parallel()
	users := newFakeUsers()
	ensure := commands.NewEnsureSuperAdmin(users)

	created, err := ensure.Handle(t.Context(), " Root@Example.com ", adminPassword)
	if err != nil || !created {
		t.Fatalf("first Handle = (%v, %v), want (true, nil)", created, err)
	}

	stored, err := users.FindByEmail(t.Context(), adminEmail)
	if err != nil {
		t.Fatalf("the super admin was not stored: %v", err)
	}
	if stored.Role != domain.RoleSuperAdmin || !stored.CheckPassword(adminPassword) {
		t.Errorf("stored = %+v, want a super admin that verifies the configured password", stored)
	}

	// A restart with a changed password must not take over the existing account.
	created, err = ensure.Handle(t.Context(), adminEmail, "a different password entirely")
	if err != nil || created {
		t.Fatalf("second Handle = (%v, %v), want (false, nil)", created, err)
	}

	again, _ := users.FindByEmail(t.Context(), adminEmail)
	if again.PasswordHash != stored.PasswordHash || !again.CheckPassword(adminPassword) {
		t.Error("the existing super admin's password was overwritten")
	}
}

func TestEnsureSuperAdminRefusesToTouchAnOwnerAccount(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	registerOwner(t, h)

	created, err := commands.NewEnsureSuperAdmin(h.users).Handle(t.Context(), ownerEmail, adminPassword)
	if !errors.Is(err, commands.ErrEmailBelongsToAnotherRole) || created {
		t.Fatalf("Handle = (%v, %v), want (false, %v)", created, err, commands.ErrEmailBelongsToAnotherRole)
	}

	owner, _ := h.users.FindByEmail(t.Context(), ownerEmail)
	if owner.Role != domain.RoleBusinessAdmin || !owner.CheckPassword(ownerPassword) {
		t.Errorf("the owner account was modified: %+v", owner)
	}
}

func TestEnsureSuperAdminRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, email, password string
		wantErr               error
	}{
		{"bad email", "nope", adminPassword, domain.ErrEmailInvalid},
		{"short password", adminEmail, "short", domain.ErrPasswordTooShort},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			users := newFakeUsers()

			if _, err := commands.NewEnsureSuperAdmin(users).Handle(t.Context(), tt.email, tt.password); !errors.Is(err, tt.wantErr) {
				t.Errorf("Handle error = %v, want %v", err, tt.wantErr)
			}
			if len(users.byEmail) != 0 {
				t.Error("an account was stored for rejected input")
			}
		})
	}
}

// racingUsers hides the existing account from the first lookup, as if another
// API instance created it between our check and our insert.
type racingUsers struct {
	*fakeUsers
	hidden bool
}

func (r *racingUsers) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	if !r.hidden {
		r.hidden = true
		return domain.User{}, identity.ErrUserNotFound
	}
	return r.fakeUsers.FindByEmail(ctx, email)
}

func TestEnsureSuperAdminToleratesAConcurrentStartup(t *testing.T) {
	t.Parallel()
	base := newFakeUsers()

	existing, err := domain.NewUser(adminEmail, adminPassword, domain.RoleSuperAdmin)
	if err != nil {
		t.Fatalf("NewUser: %v", err)
	}
	if _, err := base.Create(t.Context(), existing); err != nil {
		t.Fatalf("Create: %v", err)
	}

	created, err := commands.NewEnsureSuperAdmin(&racingUsers{fakeUsers: base}).Handle(t.Context(), adminEmail, adminPassword)
	if err != nil || created {
		t.Errorf("Handle = (%v, %v), want (false, nil) when another instance won the race", created, err)
	}
}
