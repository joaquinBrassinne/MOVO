// Package identity is the module that registers owners and super admins and
// authenticates them. This file declares the ports the application layer
// consumes; adapters implement them.
package identity

import (
	"context"
	"errors"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/identity/domain"
)

// Errors the adapters report so the application layer can react without knowing
// the storage engine.
var (
	ErrEmailTaken          = errors.New("identity: email is already registered")
	ErrUserNotFound        = errors.New("identity: user not found")
	ErrRefreshTokenInvalid = errors.New("identity: refresh token is unknown, expired or revoked")
	// ErrRefreshTokenReused means a token that was already rotated came back. The
	// store has revoked every token of that user by the time it returns this.
	ErrRefreshTokenReused = errors.New("identity: refresh token was already rotated")
)

// UserRepository stores users.
type UserRepository interface {
	// Create stores a new user and returns it with ID and CreatedAt set. It
	// returns ErrEmailTaken when the email is already registered.
	Create(ctx context.Context, user domain.User) (domain.User, error)
	// FindByEmail expects an already normalized email and returns ErrUserNotFound
	// when there is no such user.
	FindByEmail(ctx context.Context, email string) (domain.User, error)
	// FindByID returns ErrUserNotFound when there is no such user.
	FindByID(ctx context.Context, id int64) (domain.User, error)
}

// RefreshTokenStore keeps the digests of the refresh tokens.
type RefreshTokenStore interface {
	// Issue stores the digest of a first token for the user.
	Issue(ctx context.Context, userID int64, hash []byte, expiresAt time.Time) error
	// Rotate atomically revokes the presented token and stores next in its place,
	// returning the owner. Of two concurrent calls with the same token, only one
	// succeeds. A token that is unknown, expired or revoked yields
	// ErrRefreshTokenInvalid; a token that was already rotated yields
	// ErrRefreshTokenReused together with its owner, after revoking all the
	// owner's tokens.
	Rotate(ctx context.Context, presented, next []byte, nextExpiresAt, now time.Time) (userID int64, err error)
	// Revoke invalidates a token. It is idempotent: an unknown or already revoked
	// token is not an error.
	Revoke(ctx context.Context, hash []byte, now time.Time) error
}

// AccessTokenIssuer signs short-lived access tokens.
type AccessTokenIssuer interface {
	Issue(userID int64, role domain.Role, now time.Time) (token string, expiresAt time.Time, err error)
}

// LoginLimiter throttles failed logins per key (an email or a client address).
type LoginLimiter interface {
	// Allow reports whether the key may try again and, when it may not, how long
	// until it can.
	Allow(key string) (retryAfter time.Duration, allowed bool)
	// Fail records a failed attempt for the key.
	Fail(key string)
	// Reset forgets the failures of the key.
	Reset(key string)
}
