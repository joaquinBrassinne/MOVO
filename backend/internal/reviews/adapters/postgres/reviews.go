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

// FormByTokenHash reads what the review form shows about the coupon of a token: the
// business and the branch that redeemed it, the code, the name and whether it was already
// reviewed. It reads no phone. The branch is empty for a coupon that was not redeemed.
func (r *Reviews) FormByTokenHash(ctx context.Context, tokenHash []byte) (reviews.FormData, error) {
	var (
		out        reviews.FormData
		branch     *string
		city       *string
		redeemedAt *time.Time
	)
	err := r.pool.QueryRow(ctx, `
SELECT bz.name, b.name, b.city, c.code, cu.name, c.redeemed_at,
       EXISTS (SELECT 1 FROM reviews r WHERE r.benefit_code_id = c.id)
FROM benefit_codes c
JOIN businesses bz ON bz.id = c.business_id
JOIN customers cu ON cu.id = c.customer_id
LEFT JOIN branches b ON b.id = c.redeemed_branch_id
WHERE c.access_token_hash = $1`, tokenHash,
	).Scan(&out.BusinessName, &branch, &city, &out.Code, &out.CustomerName, &redeemedAt, &out.Reviewed)
	if errors.Is(err, pgx.ErrNoRows) {
		return reviews.FormData{}, reviews.ErrCouponNotFound
	}
	if err != nil {
		return reviews.FormData{}, fmt.Errorf("postgres: review form by token: %w", err)
	}
	if branch != nil && city != nil && redeemedAt != nil {
		out.BranchName, out.City, out.RedeemedAt = *branch, *city, *redeemedAt
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

// requireVisible answers ErrBranchNotFound for a branch the public cannot see, the same
// for one that does not exist.
func (r *Reviews) requireVisible(ctx context.Context, branchID int64) error {
	var visible bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM visible_branches WHERE branch_id = $1)`, branchID).Scan(&visible); err != nil {
		return fmt.Errorf("postgres: branch visibility: %w", err)
	}
	if !visible {
		return reviews.ErrBranchNotFound
	}
	return nil
}

// Summary counts the reviews the public sees by stars, with the visibility and the rule
// of ListPublic: a reported review counts and a hidden one does not.
func (r *Reviews) Summary(ctx context.Context, branchID int64) (reviews.RatingSummary, error) {
	if err := r.requireVisible(ctx, branchID); err != nil {
		return reviews.RatingSummary{}, err
	}
	rows, err := r.pool.Query(ctx, `SELECT rating, count(*) FROM reviews WHERE branch_id = $1 AND status <> 'hidden' GROUP BY rating`, branchID)
	if err != nil {
		return reviews.RatingSummary{}, fmt.Errorf("postgres: rating summary: %w", err)
	}
	defer rows.Close()

	var out reviews.RatingSummary
	for rows.Next() {
		var stars, n int
		if err := rows.Scan(&stars, &n); err != nil {
			return reviews.RatingSummary{}, fmt.Errorf("postgres: scan rating summary: %w", err)
		}
		out.Stars[stars-1] = n
		out.Count += n
	}
	if err := rows.Err(); err != nil {
		return reviews.RatingSummary{}, fmt.Errorf("postgres: rating summary: %w", err)
	}
	return out, nil
}

// ListPublic reads the reviews of a branch the public can see, newest first. A reported
// review is listed and a hidden one is not. The order is (created_at, id), which is also
// the order of the index, so the cursor of a page is the position after its last review.
// It selects no customer column.
func (r *Reviews) ListPublic(ctx context.Context, branchID int64, stars domain.StarFilter, after *domain.Cursor, limit int) ([]reviews.ListedReview, error) {
	if err := r.requireVisible(ctx, branchID); err != nil {
		return nil, err
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
  AND ($5::int = 0 OR rating >= $5::int) AND ($6::int = 0 OR rating <= $6::int)
ORDER BY created_at DESC, id DESC
LIMIT $4`, branchID, afterAt, afterID, limit, stars.Min, stars.Max)
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

// refreshRating recomputes the average and the count the cards of a branch show, from
// the reviews themselves: every one but the hidden ones, so a reported review still
// counts (SDD 5.5). Recounting instead of adding one is what makes it idempotent. The
// ratings live on branches, but they are the aggregate of this module's table, so this
// module is the one that writes them, in the transaction that changes the reviews, the
// way benefits writes the coupon limit it owns on businesses.
const refreshRating = `
UPDATE branches
SET rating_count = s.total, rating_avg = s.average
FROM (
    SELECT count(*)::int AS total, COALESCE(round(avg(rating), 2), 0) AS average
    FROM reviews WHERE branch_id = $1 AND status <> 'hidden'
) s
WHERE branches.id = $1`

// Create inserts the review and refreshes the rating of its branch in one transaction.
// The branch row is locked first: under READ COMMITTED two transactions that recount at
// once would each see only their own review and one of them would be lost, so the second
// waits for the first to commit and then counts both. A second review of the coupon
// breaks the unique key, which is also what settles two requests racing for the same
// coupon; the transaction is rolled back and the rating is not touched.
func (r *Reviews) Create(ctx context.Context, review domain.Review) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var locked int64
	if err := tx.QueryRow(ctx, `SELECT id FROM branches WHERE id = $1 FOR UPDATE`, review.BranchID).Scan(&locked); err != nil {
		return fmt.Errorf("postgres: lock branch: %w", err)
	}

	var comment *string
	if review.Comment != "" {
		comment = &review.Comment
	}
	_, err = tx.Exec(ctx, `
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

	if _, err := tx.Exec(ctx, refreshRating, review.BranchID); err != nil {
		return fmt.Errorf("postgres: refresh rating: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres: commit: %w", err)
	}
	return nil
}
