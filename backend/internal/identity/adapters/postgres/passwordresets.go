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

// PasswordResets implements identity.PasswordResetStore.
type PasswordResets struct {
	pool *pgxpool.Pool
}

// NewPasswordResets builds the store over pool.
func NewPasswordResets(pool *pgxpool.Pool) *PasswordResets {
	return &PasswordResets{pool: pool}
}

const insertAudit = `
	INSERT INTO audit_log (actor_id, action, entity, entity_id) VALUES ($1, $2, 'user', $3)`

// Issue retires the unused tokens of the user and stores the new one, with the audit entry
// of the super admin, in one transaction.
func (s *PasswordResets) Issue(ctx context.Context, actorID, userID int64, hash []byte, expiresAt, now time.Time) error {
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx,
			`UPDATE password_reset_tokens SET used_at = $2 WHERE user_id = $1 AND used_at IS NULL`, userID, now); err != nil {
			return fmt.Errorf("retire previous tokens: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO password_reset_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
			userID, hash, expiresAt); err != nil {
			return fmt.Errorf("insert token: %w", err)
		}
		if _, err := tx.Exec(ctx, insertAudit, actorID, "user.password_reset_issue", userID); err != nil {
			return fmt.Errorf("audit issue: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("postgres: issue password reset: %w", err)
	}
	return nil
}

// Redeem consumes the token, sets the password and ends every session of the owner. The
// conditional UPDATE lets exactly one of two concurrent calls match the row.
func (s *PasswordResets) Redeem(ctx context.Context, hash []byte, passwordHash string, now time.Time) (int64, error) {
	var userID int64
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			UPDATE password_reset_tokens SET used_at = $2
			WHERE token_hash = $1 AND used_at IS NULL AND expires_at > $2
			RETURNING user_id`, hash, now).Scan(&userID)
		if errors.Is(err, pgx.ErrNoRows) {
			return identity.ErrResetTokenInvalid
		}
		if err != nil {
			return fmt.Errorf("consume token: %w", err)
		}
		if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2 WHERE id = $1`, userID, passwordHash); err != nil {
			return fmt.Errorf("set password: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`UPDATE refresh_tokens SET revoked_at = $2, updated_at = $2 WHERE user_id = $1 AND revoked_at IS NULL`,
			userID, now); err != nil {
			return fmt.Errorf("revoke sessions: %w", err)
		}
		if _, err := tx.Exec(ctx, insertAudit, userID, "user.password_reset", userID); err != nil {
			return fmt.Errorf("audit redeem: %w", err)
		}
		return nil
	})
	if errors.Is(err, identity.ErrResetTokenInvalid) {
		return 0, err
	}
	if err != nil {
		return 0, fmt.Errorf("postgres: redeem password reset: %w", err)
	}
	return userID, nil
}
