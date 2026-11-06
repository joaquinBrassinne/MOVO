// Package commands holds the identity use cases that change state: Register,
// Login, Refresh and Logout.
package commands

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/identity"
	"github.com/joaquinBrassinne/MOVO/internal/identity/domain"
)

// ErrSessionInvalid is returned for every refresh token that cannot be used,
// whatever the reason, so a caller learns nothing about which tokens exist.
var ErrSessionInvalid = errors.New("identity: session is not valid")

// Session is what a successful Register, Login or Refresh hands back.
type Session struct {
	User             domain.User
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

// Sessions issues and rotates the token pairs of the use cases.
type Sessions struct {
	access     identity.AccessTokenIssuer
	store      identity.RefreshTokenStore
	users      identity.UserRepository
	refreshTTL time.Duration
	now        func() time.Time
	log        *slog.Logger
}

// NewSessions wires the token collaborators. A nil now uses time.Now and a nil
// logger discards logs.
func NewSessions(
	access identity.AccessTokenIssuer,
	store identity.RefreshTokenStore,
	users identity.UserRepository,
	refreshTTL time.Duration,
	now func() time.Time,
	log *slog.Logger,
) *Sessions {
	if now == nil {
		now = time.Now
	}
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}

	return &Sessions{access: access, store: store, users: users, refreshTTL: refreshTTL, now: now, log: log}
}

// open starts a session for user with a brand new refresh token.
func (s *Sessions) open(ctx context.Context, user domain.User) (Session, error) {
	now := s.now()

	plain, hash, err := domain.NewRefreshToken()
	if err != nil {
		return Session{}, err
	}

	expires := now.Add(s.refreshTTL)
	if err := s.store.Issue(ctx, user.ID, hash, expires); err != nil {
		return Session{}, fmt.Errorf("store refresh token: %w", err)
	}

	return s.pair(user, plain, expires, now)
}

// rotate trades a presented refresh token for a new pair.
func (s *Sessions) rotate(ctx context.Context, presented string) (Session, error) {
	if presented == "" {
		return Session{}, ErrSessionInvalid
	}

	now := s.now()

	plain, hash, err := domain.NewRefreshToken()
	if err != nil {
		return Session{}, err
	}

	expires := now.Add(s.refreshTTL)
	userID, err := s.store.Rotate(ctx, domain.HashRefreshToken(presented), hash, expires, now)
	switch {
	case errors.Is(err, identity.ErrRefreshTokenReused):
		s.log.Warn("refresh token reuse detected, all sessions of the user were revoked", "user_id", userID)
		return Session{}, ErrSessionInvalid
	case errors.Is(err, identity.ErrRefreshTokenInvalid):
		return Session{}, ErrSessionInvalid
	case err != nil:
		return Session{}, fmt.Errorf("rotate refresh token: %w", err)
	}

	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return Session{}, fmt.Errorf("load session user: %w", err)
	}

	return s.pair(user, plain, expires, now)
}

func (s *Sessions) pair(user domain.User, refresh string, refreshExpires, now time.Time) (Session, error) {
	access, accessExpires, err := s.access.Issue(user.ID, user.Role, now)
	if err != nil {
		return Session{}, fmt.Errorf("issue access token: %w", err)
	}

	return Session{
		User:             user,
		AccessToken:      access,
		AccessExpiresAt:  accessExpires,
		RefreshToken:     refresh,
		RefreshExpiresAt: refreshExpires,
	}, nil
}
