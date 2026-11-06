package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/identity"
)

// RefreshTokens implements identity.RefreshTokenStore.
type RefreshTokens struct {
	pool *pgxpool.Pool
}

// NewRefreshTokens builds the store over pool.
func NewRefreshTokens(pool *pgxpool.Pool) *RefreshTokens {
	return &RefreshTokens{pool: pool}
}

const insertToken = `
	INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3) RETURNING id`

// Issue stores the digest of a first token for the user.
func (s *RefreshTokens) Issue(ctx context.Context, userID int64, hash []byte, expiresAt time.Time) error {
	if _, err := s.pool.Exec(ctx, insertToken, userID, hash, expiresAt); err != nil {
		return fmt.Errorf("postgres: insert refresh token: %w", err)
	}
	return nil
}

// Rotate revokes the presented token and stores next in its place, in one
// transaction. The conditional UPDATE is what makes concurrent rotations of the
// same token race-safe: PostgreSQL lets exactly one of them match the row.
func (s *RefreshTokens) Rotate(
	ctx context.Context,
	presented, next []byte,
	nextExpiresAt, now time.Time,
) (int64, error) {
	var (
		userID int64
		result error
	)

	// The reuse branch must commit its revocation, so it reports through result
	// instead of returning an error that would roll the transaction back.
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var oldID int64
		err := tx.QueryRow(ctx, `
			UPDATE refresh_tokens
			SET revoked_at = $2, updated_at = $2
			WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > $2
			RETURNING id, user_id`, presented, now).Scan(&oldID, &userID)

		if errors.Is(err, pgx.ErrNoRows) {
			userID, result = s.rejectRotation(ctx, tx, presented, now)
			return nil
		}
		if err != nil {
			return fmt.Errorf("revoke presented token: %w", err)
		}

		var newID int64
		if err := tx.QueryRow(ctx, insertToken, userID, next, nextExpiresAt).Scan(&newID); err != nil {
			return fmt.Errorf("insert next token: %w", err)
		}

		if _, err := tx.Exec(ctx, `UPDATE refresh_tokens SET replaced_by = $2 WHERE id = $1`, oldID, newID); err != nil {
			return fmt.Errorf("link rotated token: %w", err)
		}

		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("postgres: rotate refresh token: %w", err)
	}

	return userID, result
}

// rejectRotation explains why nothing matched. A token that was already rotated
// is a replay, so every live token of its owner is revoked.
func (s *RefreshTokens) rejectRotation(ctx context.Context, tx pgx.Tx, presented []byte, now time.Time) (int64, error) {
	var (
		owner   int64
		rotated bool
	)

	err := tx.QueryRow(ctx,
		`SELECT user_id, replaced_by IS NOT NULL FROM refresh_tokens WHERE token_hash = $1`, presented,
	).Scan(&owner, &rotated)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("find rejected token: %w", err)
	}
	if err != nil || !rotated {
		// Unknown, expired, or revoked by a logout: nothing suspicious to punish.
		return 0, identity.ErrRefreshTokenInvalid
	}

	_, err = tx.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = $2, updated_at = $2 WHERE user_id = $1 AND revoked_at IS NULL`,
		owner, now)
	if err != nil {
		return 0, fmt.Errorf("revoke token family: %w", err)
	}

	return owner, identity.ErrRefreshTokenReused
}

// Revoke invalidates a token; unknown or already revoked tokens are ignored.
func (s *RefreshTokens) Revoke(ctx context.Context, hash []byte, now time.Time) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = $2, updated_at = $2 WHERE token_hash = $1 AND revoked_at IS NULL`,
		hash, now)
	if err != nil {
		return fmt.Errorf("postgres: revoke refresh token: %w", err)
	}
	return nil
}
