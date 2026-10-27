package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

// SetAdminPaused sets the super admin's pause on any benefit and records who did it, in
// one transaction: no pause without its audit entry. Repeating the current state changes
// nothing and records nothing.
func (r *Benefits) SetAdminPaused(ctx context.Context, actorID, benefitID int64, paused bool) (domain.Benefit, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Benefit{}, fmt.Errorf("postgres: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	b, err := scanBenefit(tx.QueryRow(ctx, `
UPDATE benefits SET admin_paused = $2, updated_at = now()
WHERE id = $1 AND admin_paused <> $2
RETURNING `+benefitColumns(""), benefitID, paused))
	if errors.Is(err, pgx.ErrNoRows) {
		// Either it does not exist or it already is in that state.
		current, err := scanBenefit(tx.QueryRow(ctx, selectBenefit+` WHERE id = $1`, benefitID))
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Benefit{}, benefits.ErrBenefitNotFound
		}
		return current, err
	}
	if err != nil {
		return domain.Benefit{}, err
	}

	action := "benefit.pause"
	if !paused {
		action = "benefit.resume"
	}
	metadata, err := json.Marshal(map[string]any{"business_id": b.BusinessID})
	if err != nil {
		return domain.Benefit{}, fmt.Errorf("postgres: encode audit metadata: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO audit_log (actor_id, action, entity, entity_id, metadata) VALUES ($1, $2, 'benefit', $3, $4)`,
		actorID, action, b.ID, metadata); err != nil {
		return domain.Benefit{}, fmt.Errorf("postgres: append audit entry: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Benefit{}, fmt.Errorf("postgres: commit: %w", err)
	}
	return b, nil
}
