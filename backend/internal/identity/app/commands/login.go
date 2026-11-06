package commands

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/identity"
	"github.com/joaquinBrassinne/MOVO/internal/identity/domain"
)

// ErrInvalidCredentials covers an unknown email and a wrong password alike, so
// the response never says which accounts exist.
var ErrInvalidCredentials = errors.New("identity: invalid credentials")

// RateLimitedError reports that too many logins failed recently.
type RateLimitedError struct {
	RetryAfter time.Duration
}

func (e *RateLimitedError) Error() string {
	return fmt.Sprintf("identity: too many failed logins, retry in %s", e.RetryAfter.Round(time.Second))
}

// dummyHash is verified when the email is unknown, so that case costs as much
// as a wrong password and response time does not reveal which emails exist.
var dummyHash = sync.OnceValue(func() string {
	hash, err := domain.HashPassword("never-matches-a-real-account")
	if err != nil {
		return ""
	}
	return hash
})

// LoginInput carries the credentials and the caller's address, which only feeds
// the rate limiter and is never stored.
type LoginInput struct {
	Email    string
	Password string
	ClientIP string
}

// Login authenticates a user by email and password.
type Login struct {
	users    identity.UserRepository
	limiter  identity.LoginLimiter
	sessions *Sessions
}

// NewLogin wires the Login use case.
func NewLogin(users identity.UserRepository, limiter identity.LoginLimiter, sessions *Sessions) *Login {
	return &Login{users: users, limiter: limiter, sessions: sessions}
}

// Handle checks the rate limit, verifies the credentials and opens a session.
func (c *Login) Handle(ctx context.Context, in LoginInput) (Session, error) {
	email, emailErr := domain.NormalizeEmail(in.Email)

	var keys []string
	if in.ClientIP != "" {
		keys = append(keys, "ip:"+in.ClientIP)
	}
	if emailErr == nil {
		keys = append(keys, "email:"+email)
	}

	var wait time.Duration
	for _, key := range keys {
		if retry, allowed := c.limiter.Allow(key); !allowed {
			wait = max(wait, retry)
		}
	}
	if wait > 0 {
		return Session{}, &RateLimitedError{RetryAfter: wait}
	}

	user, err := c.authenticate(ctx, email, emailErr, in.Password)
	if errors.Is(err, ErrInvalidCredentials) {
		for _, key := range keys {
			c.limiter.Fail(key)
		}
		return Session{}, err
	}
	if err != nil {
		return Session{}, err
	}

	// Only the account key is cleared: resetting the address key would let an
	// attacker with an account of their own wipe their failures at will.
	c.limiter.Reset("email:" + email)

	return c.sessions.open(ctx, user)
}

func (c *Login) authenticate(ctx context.Context, email string, emailErr error, password string) (domain.User, error) {
	if emailErr != nil {
		_, _ = domain.VerifyPassword(dummyHash(), password)
		return domain.User{}, ErrInvalidCredentials
	}

	user, err := c.users.FindByEmail(ctx, email)
	if errors.Is(err, identity.ErrUserNotFound) {
		_, _ = domain.VerifyPassword(dummyHash(), password)
		return domain.User{}, ErrInvalidCredentials
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("find user: %w", err)
	}

	if !user.CheckPassword(password) {
		return domain.User{}, ErrInvalidCredentials
	}

	return user, nil
}
