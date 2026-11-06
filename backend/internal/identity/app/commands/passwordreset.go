package commands

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/identity"
	"github.com/joaquinBrassinne/MOVO/internal/identity/domain"
)

// ResetLinkTTL is how long an assisted reset link works. The super admin sends it by
// WhatsApp, so the owner may take a while to read it.
const ResetLinkTTL = 24 * time.Hour

// IssuedReset is a freshly issued link. Token is shown once and never stored in plain.
type IssuedReset struct {
	UserID    int64
	Email     string
	Token     string
	ExpiresAt time.Time
}

// IssuePasswordReset lets a super admin issue a one-time reset token for an owner.
type IssuePasswordReset struct {
	users identity.UserRepository
	store identity.PasswordResetStore
	now   func() time.Time
}

// NewIssuePasswordReset wires the use case. A nil now uses time.Now.
func NewIssuePasswordReset(users identity.UserRepository, store identity.PasswordResetStore, now func() time.Time) *IssuePasswordReset {
	if now == nil {
		now = time.Now
	}
	return &IssuePasswordReset{users: users, store: store, now: now}
}

// Handle issues a token for the owner with that email. Only business owners can be reset:
// a super admin account answers identity.ErrUserNotFound like an unknown email, so this
// path can never be used to take over an admin.
func (c *IssuePasswordReset) Handle(ctx context.Context, actorID int64, email string) (IssuedReset, error) {
	normalized, err := domain.NormalizeEmail(email)
	if err != nil {
		return IssuedReset{}, err
	}
	user, err := c.users.FindByEmail(ctx, normalized)
	if err != nil {
		return IssuedReset{}, fmt.Errorf("find owner: %w", err)
	}
	if user.Role != domain.RoleBusinessAdmin {
		return IssuedReset{}, identity.ErrUserNotFound
	}

	plain, hash, err := domain.NewRefreshToken()
	if err != nil {
		return IssuedReset{}, err
	}
	now := c.now()
	expires := now.Add(ResetLinkTTL)
	if err := c.store.Issue(ctx, actorID, user.ID, hash, expires, now); err != nil {
		return IssuedReset{}, fmt.Errorf("store reset token: %w", err)
	}
	return IssuedReset{UserID: user.ID, Email: user.Email, Token: plain, ExpiresAt: expires}, nil
}

// ResetPasswordInput is the owner opening the link. ClientIP only feeds the rate limiter.
type ResetPasswordInput struct {
	Token    string
	Password string
	ClientIP string
}

// ResetPassword redeems a reset token and sets the new password.
type ResetPassword struct {
	store   identity.PasswordResetStore
	limiter identity.RegistrationLimiter
	now     func() time.Time
}

// NewResetPassword wires the use case. A nil now uses time.Now.
func NewResetPassword(store identity.PasswordResetStore, limiter identity.RegistrationLimiter, now func() time.Time) *ResetPassword {
	if now == nil {
		now = time.Now
	}
	return &ResetPassword{store: store, limiter: limiter, now: now}
}

// Handle counts the try against the address first, because hashing is the expensive step,
// then validates the password and redeems the token. Every token that cannot be used is
// identity.ErrResetTokenInvalid, whatever the reason. All the owner's sessions end.
func (c *ResetPassword) Handle(ctx context.Context, in ResetPasswordInput) error {
	if in.ClientIP != "" {
		if retry, allowed := c.limiter.Attempt("reset:" + in.ClientIP); !allowed {
			return &RateLimitedError{RetryAfter: retry}
		}
	}
	if in.Token == "" {
		return identity.ErrResetTokenInvalid
	}
	if err := domain.ValidatePassword(in.Password); err != nil {
		return err
	}
	hash, err := domain.HashPassword(in.Password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if _, err := c.store.Redeem(ctx, domain.HashRefreshToken(in.Token), hash, c.now()); err != nil {
		if errors.Is(err, identity.ErrResetTokenInvalid) {
			return err
		}
		return fmt.Errorf("redeem reset token: %w", err)
	}
	return nil
}
