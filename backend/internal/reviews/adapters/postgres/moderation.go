package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/joaquinBrassinne/MOVO/internal/reviews"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

// appendAudit records who did what to a review, in the transaction of the change itself:
// no change without its audit entry.
func appendAudit(ctx context.Context, tx pgx.Tx, actorID int64, action string, reviewID int64, metadata map[string]any) error {
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("postgres: encode audit metadata: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO audit_log (actor_id, action, entity, entity_id, metadata) VALUES ($1, $2, 'review', $3, $4)`,
		actorID, action, reviewID, encoded); err != nil {
		return fmt.Errorf("postgres: append audit entry: %w", err)
	}
	return nil
}

// Report marks a visible review of the business as reported and records it, in one
// transaction. The business is part of the WHERE, so a review of another business is
// indistinguishable from one that does not exist. A review that is already reported or
// hidden is left as it is and nothing is recorded. The rating is not touched: a reported
// review still counts.
func (r *Reviews) Report(ctx context.Context, actorID, businessID, reviewID int64) (reviews.ModeratedReview, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return reviews.ModeratedReview{}, fmt.Errorf("postgres: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var branchID int64
	var status string
	err = tx.QueryRow(ctx, `
UPDATE reviews SET status = 'reported', updated_at = now()
WHERE id = $1 AND business_id = $2 AND status = 'visible'
RETURNING branch_id, status`, reviewID, businessID).Scan(&branchID, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		// Either it is not this business's, or it is not visible any more.
		if err := tx.QueryRow(ctx, `SELECT status FROM reviews WHERE id = $1 AND business_id = $2`, reviewID, businessID).Scan(&status); errors.Is(err, pgx.ErrNoRows) {
			return reviews.ModeratedReview{}, reviews.ErrReviewNotFound
		} else if err != nil {
			return reviews.ModeratedReview{}, fmt.Errorf("postgres: read review: %w", err)
		}
		return reviews.ModeratedReview{ID: reviewID, Status: domain.Status(status)}, nil
	}
	if err != nil {
		return reviews.ModeratedReview{}, fmt.Errorf("postgres: report review: %w", err)
	}

	if err := appendAudit(ctx, tx, actorID, "review.report", reviewID, map[string]any{"business_id": businessID, "branch_id": branchID}); err != nil {
		return reviews.ModeratedReview{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return reviews.ModeratedReview{}, fmt.Errorf("postgres: commit: %w", err)
	}
	return reviews.ModeratedReview{ID: reviewID, Status: domain.StatusReported}, nil
}

// SetStatus hides a review or makes it visible again, recounts the rating of its branch
// and records it, in one transaction. The branch row is locked first, in the same order
// Create locks it, so two changes to the reviews of a branch never recount without each
// other's: the second waits and then counts both. Setting the status it already has
// changes nothing and records nothing.
func (r *Reviews) SetStatus(ctx context.Context, actorID, reviewID int64, status domain.Status) (reviews.ModeratedReview, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return reviews.ModeratedReview{}, fmt.Errorf("postgres: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var branchID, businessID int64
	err = tx.QueryRow(ctx, `SELECT branch_id, business_id FROM reviews WHERE id = $1`, reviewID).Scan(&branchID, &businessID)
	if errors.Is(err, pgx.ErrNoRows) {
		return reviews.ModeratedReview{}, reviews.ErrReviewNotFound
	}
	if err != nil {
		return reviews.ModeratedReview{}, fmt.Errorf("postgres: read review: %w", err)
	}

	var locked int64
	if err := tx.QueryRow(ctx, `SELECT id FROM branches WHERE id = $1 FOR UPDATE`, branchID).Scan(&locked); err != nil {
		return reviews.ModeratedReview{}, fmt.Errorf("postgres: lock branch: %w", err)
	}
	var current string
	if err := tx.QueryRow(ctx, `SELECT status FROM reviews WHERE id = $1 FOR UPDATE`, reviewID).Scan(&current); err != nil {
		return reviews.ModeratedReview{}, fmt.Errorf("postgres: lock review: %w", err)
	}
	if current == string(status) {
		return reviews.ModeratedReview{ID: reviewID, Status: status}, nil
	}

	if _, err := tx.Exec(ctx, `UPDATE reviews SET status = $2, updated_at = now() WHERE id = $1`, reviewID, string(status)); err != nil {
		return reviews.ModeratedReview{}, fmt.Errorf("postgres: set review status: %w", err)
	}
	if _, err := tx.Exec(ctx, refreshRating, branchID); err != nil {
		return reviews.ModeratedReview{}, fmt.Errorf("postgres: refresh rating: %w", err)
	}
	action := "review.restore"
	if status == domain.StatusHidden {
		action = "review.hide"
	}
	if err := appendAudit(ctx, tx, actorID, action, reviewID, map[string]any{"business_id": businessID, "branch_id": branchID, "from": current}); err != nil {
		return reviews.ModeratedReview{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return reviews.ModeratedReview{}, fmt.Errorf("postgres: commit: %w", err)
	}
	return reviews.ModeratedReview{ID: reviewID, Status: status}, nil
}
