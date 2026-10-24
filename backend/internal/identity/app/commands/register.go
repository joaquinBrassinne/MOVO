package commands

import (
	"context"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/identity"
	"github.com/joaquinBrassinne/MOVO/internal/identity/domain"
)

// RegisterInput is a public sign-up. There is deliberately no role: this path
// can only ever create a business owner.
type RegisterInput struct {
	Email    string
	Password string
}

// Register creates a business owner account and signs it in.
type Register struct {
	users    identity.UserRepository
	sessions *Sessions
}

// NewRegister wires the Register use case.
func NewRegister(users identity.UserRepository, sessions *Sessions) *Register {
	return &Register{users: users, sessions: sessions}
}

// Handle validates the input, stores the user and opens a session. Validation
// failures are the domain errors; a taken email is identity.ErrEmailTaken.
func (c *Register) Handle(ctx context.Context, in RegisterInput) (Session, error) {
	user, err := domain.NewUser(in.Email, in.Password, domain.RoleBusinessAdmin)
	if err != nil {
		return Session{}, err
	}

	stored, err := c.users.Create(ctx, user)
	if err != nil {
		return Session{}, fmt.Errorf("create user: %w", err)
	}

	return c.sessions.open(ctx, stored)
}
