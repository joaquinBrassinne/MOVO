// Package postgres stores reviews and finds the redemption a customer reviews.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/reviews"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

const uniqueViolation = "23505"

// Reviews reads redemptions and stores reviews.
type Reviews struct {
	pool *pgxpool.Pool
}

// NewReviews returns a Reviews backed by pool.
func NewReviews(pool *pgxpool.Pool) *Reviews {
	return &Reviews{pool: pool}
}

// ByTokenHash reads the coupon whose access token has that hash. It is a plain SELECT
// on the unique hash, so it can only return that one coupon, and it selects no phone
// column. A coupon that is not redeemed has no branch nor time, and comes back with a
// zero RedeemedAt.
func (r *Reviews) ByTokenHash(ctx context.Context, tokenHash []byte) (domain.Redemption, error) {
	var (
		out        domain.Redemption
		branchID   *int64
		redeemedAt *time.Time
	)
	err := r.pool.QueryRow(ctx, `
SELECT c.id, c.business_id, c.customer_id, cu.name, c.redeemed_branch_id, c.redeemed_at
FROM benefit_codes c
JOIN customers cu ON cu.id = c.customer_id
WHERE c.access_token_hash = $1 AND c.status = 'redeemed'`, tokenHash,
	).Scan(&out.CouponID, &out.BusinessID, &out.CustomerID, &out.CustomerName, &branchID, &redeemedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return r.unredeemed(ctx, tokenHash)
	}
	if err != nil {
		return domain.Redemption{}, fmt.Errorf("postgres: redemption by token: %w", err)
	}
	if branchID != nil && redeemedAt != nil {
		out.BranchID, out.RedeemedAt = *branchID, *redeemedAt
	}
	return out, nil
}

// unredeemed tells a coupon that exists but was not redeemed (a zero Redemption the
// domain refuses) from a token that opens nothing.
func (r *Reviews) unredeemed(ctx context.Context, tokenHash []byte) (domain.Redemption, error) {
	var exists bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM benefit_codes WHERE access_token_hash = $1)`, tokenHash).Scan(&exists); err != nil {
		return domain.Redemption{}, fmt.Errorf("postgres: coupon by token: %w", err)
	}
	if !exists {
		return domain.Redemption{}, reviews.ErrCouponNotFound
	}
	return domain.Redemption{}, nil
}

// ListPublic reads the reviews of a branch the public can see, newest first. A reported
// review is listed and a hidden one is not. The order is (created_at, id), which is also
// the order of the index, so the cursor of a page is the position after its last review.
// It selects no customer column.
func (r *Reviews) ListPublic(ctx context.Context, branchID int64, after *domain.Cursor, limit int) ([]reviews.ListedReview, error) {
	var visible bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM visible_branches WHERE branch_id = $1)`, branchID).Scan(&visible); err != nil {
		return nil, fmt.Errorf("postgres: branch visibility: %w", err)
	}
	if !visible {
		return nil, reviews.ErrBranchNotFound
	}

	var (
		afterAt *time.Time
		afterID *int64
	)
	if after != nil {
		afterAt, afterID = &after.CreatedAt, &after.ID
	}
	rows, err := r.pool.Query(ctx, `
SELECT id, rating, comment, display_name, created_at
FROM reviews
WHERE branch_id = $1 AND status <> 'hidden'
  AND ($2::timestamptz IS NULL OR (created_at, id) < ($2, $3::bigint))
ORDER BY created_at DESC, id DESC
LIMIT $4`, branchID, afterAt, afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("postgres: list reviews: %w", err)
	}
	defer rows.Close()

	var out []reviews.ListedReview
	for rows.Next() {
		var (
			item    reviews.ListedReview
			comment *string
		)
		if err := rows.Scan(&item.ID, &item.Rating, &comment, &item.DisplayName, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("postgres: scan review: %w", err)
		}
		if comment != nil {
			item.Comment = *comment
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: list reviews: %w", err)
	}
	return out, nil
}

// Create inserts the review. A second review of the coupon breaks the unique key, which
// is also what settles two requests racing for the same coupon.
func (r *Reviews) Create(ctx context.Context, review domain.Review) error {
	var comment *string
	if review.Comment != "" {
		comment = &review.Comment
	}
	_, err := r.pool.Exec(ctx, `
INSERT INTO reviews (branch_id, business_id, benefit_code_id, customer_id, rating, comment, display_name, show_name, status, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		review.BranchID, review.BusinessID, review.CouponID, review.CustomerID, review.Rating, comment,
		review.DisplayName, review.ShowName, string(review.Status), review.CreatedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return reviews.ErrAlreadyReviewed
	}
	if err != nil {
		return fmt.Errorf("postgres: insert review: %w", err)
	}
	return nil
}
