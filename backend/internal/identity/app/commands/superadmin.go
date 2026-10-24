package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/identity"
	"github.com/joaquinBrassinne/MOVO/internal/identity/domain"
)

// ErrEmailBelongsToAnotherRole means the configured super admin email is already
// registered as something else. Promoting a real owner by configuration would
// be a silent privilege change, so it is refused.
var ErrEmailBelongsToAnotherRole = errors.New("identity: the super admin email is registered with another role")

// EnsureSuperAdmin makes sure the configured super admin account exists.
type EnsureSuperAdmin struct {
	users identity.UserRepository
}

// NewEnsureSuperAdmin wires the seed use case.
func NewEnsureSuperAdmin(users identity.UserRepository) *EnsureSuperAdmin {
	return &EnsureSuperAdmin{users: users}
}

// Handle creates the super admin when the email is unknown and reports whether it
// did. It is idempotent and never changes an existing account, so restarting
// with another password in the environment does not take it over; a password
// change goes through the super admin's own reset flow.
func (c *EnsureSuperAdmin) Handle(ctx context.Context, email, password string) (created bool, err error) {
	normalized, err := domain.NormalizeEmail(email)
	if err != nil {
		return false, err
	}

	existing, err := c.users.FindByEmail(ctx, normalized)
	if err == nil {
		return false, requireSuperAdmin(existing)
	}
	if !errors.Is(err, identity.ErrUserNotFound) {
		return false, fmt.Errorf("find super admin: %w", err)
	}

	user, err := domain.NewUser(normalized, password, domain.RoleSuperAdmin)
	if err != nil {
		return false, err
	}

	if _, err := c.users.Create(ctx, user); err != nil {
		if !errors.Is(err, identity.ErrEmailTaken) {
			return false, fmt.Errorf("create super admin: %w", err)
		}

		// Another instance created it between our lookup and our insert.
		existing, err := c.users.FindByEmail(ctx, normalized)
		if err != nil {
			return false, fmt.Errorf("find super admin after conflict: %w", err)
		}
		return false, requireSuperAdmin(existing)
	}

	return true, nil
}

func requireSuperAdmin(user domain.User) error {
	if user.Role != domain.RoleSuperAdmin {
		return ErrEmailBelongsToAnotherRole
	}
	return nil
}
