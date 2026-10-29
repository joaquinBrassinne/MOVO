package commands

import (
	"context"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/identity"
	"github.com/joaquinBrassinne/MOVO/internal/identity/domain"
)

// RegisterInput is a public sign-up. There is deliberately no role: this path
// can only ever create a business owner. ClientIP only feeds the rate limiter and
// is never stored.
type RegisterInput struct {
	Email    string
	Password string
	ClientIP string
}

// Register creates a business owner account and signs it in.
type Register struct {
	users    identity.UserRepository
	limiter  identity.RegistrationLimiter
	sessions *Sessions
}

// NewRegister wires the Register use case.
func NewRegister(users identity.UserRepository, limiter identity.RegistrationLimiter, sessions *Sessions) *Register {
	return &Register{users: users, limiter: limiter, sessions: sessions}
}

// Handle counts the try against the address, validates the input, stores the user and
// opens a session. It returns *RateLimitedError when the address spent its budget;
// that check comes first because hashing the password is the expensive step.
// Validation failures are the domain errors; a taken email is identity.ErrEmailTaken.
func (c *Register) Handle(ctx context.Context, in RegisterInput) (Session, error) {
	if in.ClientIP != "" {
		if retry, allowed := c.limiter.Attempt("ip:" + in.ClientIP); !allowed {
			return Session{}, &RateLimitedError{RetryAfter: retry}
		}
	}

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
