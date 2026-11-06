package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/reviews"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

// moderationSelect is what both moderation lists read. The caller adds its own condition
// ($1) and the keyset cursor ($2, $3), then the limit ($4).
const moderationSelect = `
SELECT r.id, r.branch_id, b.name, r.business_id, bz.name, r.rating, r.comment, r.display_name, r.status, r.created_at
FROM reviews r
JOIN branches b ON b.id = r.branch_id
JOIN businesses bz ON bz.id = r.business_id
WHERE `

const moderationPaging = `
  AND ($2::timestamptz IS NULL OR (r.created_at, r.id) < ($2, $3::bigint))
ORDER BY r.created_at DESC, r.id DESC
LIMIT $4`

// ListOfBusiness returns the visible and reported reviews of the business, newest first.
func (r *Reviews) ListOfBusiness(ctx context.Context, businessID int64, after *domain.Cursor, limit int) ([]reviews.ModerationReview, error) {
	return r.listModeration(ctx, `r.business_id = $1 AND r.status <> 'hidden'`+moderationPaging, businessID, after, limit)
}

// ListByStatus returns the reviews in that status, newest first, leaving out the ones
// whose text was purged.
func (r *Reviews) ListByStatus(ctx context.Context, status domain.Status, after *domain.Cursor, limit int) ([]reviews.ModerationReview, error) {
	return r.listModeration(ctx, `r.status = $1 AND r.purged_at IS NULL`+moderationPaging, string(status), after, limit)
}

func (r *Reviews) listModeration(ctx context.Context, where string, subject any, after *domain.Cursor, limit int) ([]reviews.ModerationReview, error) {
	var (
		afterAt *time.Time
		afterID *int64
	)
	if after != nil {
		afterAt, afterID = &after.CreatedAt, &after.ID
	}
	rows, err := r.pool.Query(ctx, moderationSelect+where, subject, afterAt, afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("postgres: list reviews to moderate: %w", err)
	}
	defer rows.Close()

	var out []reviews.ModerationReview
	for rows.Next() {
		var (
			item    reviews.ModerationReview
			comment *string
			status  string
		)
		if err := rows.Scan(&item.ID, &item.BranchID, &item.BranchName, &item.BusinessID, &item.BusinessName,
			&item.Rating, &comment, &item.DisplayName, &status, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("postgres: scan review: %w", err)
		}
		if comment != nil {
			item.Comment = *comment
		}
		item.Status = domain.Status(status)
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: list reviews to moderate: %w", err)
	}
	return out, nil
}
