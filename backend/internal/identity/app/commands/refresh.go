package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/identity"
	"github.com/joaquinBrassinne/MOVO/internal/identity/domain"
)

// Refresh trades a refresh token for a new token pair; the old token dies.
type Refresh struct {
	sessions *Sessions
}

// NewRefresh wires the Refresh use case.
func NewRefresh(sessions *Sessions) *Refresh {
	return &Refresh{sessions: sessions}
}

// Handle returns ErrSessionInvalid for any token that cannot be used.
func (c *Refresh) Handle(ctx context.Context, refreshToken string) (Session, error) {
	return c.sessions.rotate(ctx, refreshToken)
}

// Logout ends a session by revoking its refresh token.
type Logout struct {
	store identity.RefreshTokenStore
	now   func() time.Time
}

// NewLogout wires the Logout use case. A nil now uses time.Now.
func NewLogout(store identity.RefreshTokenStore, now func() time.Time) *Logout {
	if now == nil {
		now = time.Now
	}
	return &Logout{store: store, now: now}
}

// Handle is idempotent: logging out an unknown or already ended session succeeds.
func (c *Logout) Handle(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}

	if err := c.store.Revoke(ctx, domain.HashRefreshToken(refreshToken), c.now()); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	return nil
}
