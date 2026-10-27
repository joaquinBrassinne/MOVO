package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

const (
	uniqueViolation = "23505"

	codeKey       = "benefit_codes_code_key"
	onePendingKey = "benefit_codes_one_pending_idx"
)

// Coupons implements benefits.CouponIssuer.
type Coupons struct {
	pool *pgxpool.Pool
}

// NewCoupons builds the issuer over pool.
func NewCoupons(pool *pgxpool.Pool) *Coupons {
	return &Coupons{pool: pool}
}

// Issue generates one coupon in a single transaction. The benefit row is locked, so
// the monthly count and the insert of two simultaneous requests run one after the
// other and the limit cannot be overshot. The unique indexes still guard the two rules
// the database owns: the code and one pending coupon per customer and business.
func (r *Coupons) Issue(ctx context.Context, req benefits.IssueRequest) (benefits.Issued, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return benefits.Issued{}, fmt.Errorf("postgres: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Only a benefit the owner has on and the admin has not paused, of a business with a
	// visible branch, can be claimed.
	benefit, err := scanBenefit(tx.QueryRow(ctx, selectBenefit+`
WHERE id = $1 AND active AND NOT admin_paused
  AND EXISTS (SELECT 1 FROM visible_branches v WHERE v.business_id = benefits.business_id)
FOR UPDATE`, req.BenefitID))
	if errors.Is(err, pgx.ErrNoRows) {
		return benefits.Issued{}, benefits.ErrBenefitNotFound
	}
	if err != nil {
		return benefits.Issued{}, err
	}

	// One person is one phone: a repeat visit reuses the customer and keeps the name
	// they declared last.
	var customerID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO customers (name, phone) VALUES ($1, $2)
ON CONFLICT (phone) DO UPDATE SET name = EXCLUDED.name, updated_at = now()
RETURNING id`, req.Contact.Name, req.Contact.Phone).Scan(&customerID); err != nil {
		return benefits.Issued{}, fmt.Errorf("postgres: upsert customer: %w", err)
	}

	// An overdue coupon that the daily sweep has not marked yet must not block the
	// customer: it is not redeemable, so free its slot now.
	if _, err := tx.Exec(ctx, `
UPDATE benefit_codes SET status = 'expired', updated_at = now()
WHERE customer_id = $1 AND business_id = $2 AND status = 'pending' AND expires_at <= $3`,
		customerID, benefit.BusinessID, req.Now); err != nil {
		return benefits.Issued{}, fmt.Errorf("postgres: expire overdue coupons: %w", err)
	}
	var pending bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM benefit_codes WHERE customer_id = $1 AND business_id = $2 AND status = 'pending')`,
		customerID, benefit.BusinessID).Scan(&pending); err != nil {
		return benefits.Issued{}, fmt.Errorf("postgres: check pending coupon: %w", err)
	}
	if pending {
		return benefits.Issued{}, benefits.ErrPendingCoupon
	}

	var generated int
	if err := tx.QueryRow(ctx,
		`SELECT count(*) FROM benefit_codes WHERE benefit_id = $1 AND created_at >= $2 AND created_at < $3`,
		benefit.ID, req.MonthStart, req.MonthEnd).Scan(&generated); err != nil {
		return benefits.Issued{}, fmt.Errorf("postgres: count coupons of the month: %w", err)
	}
	if !benefit.HasRoom(generated) {
		return benefits.Issued{}, benefits.ErrQuotaReached
	}

	coupon := domain.Coupon{
		BenefitID: benefit.ID, BusinessID: benefit.BusinessID, CustomerID: customerID,
		Code: req.Code, Status: domain.StatusPending, ExpiresAt: req.ExpiresAt,
	}
	err = tx.QueryRow(ctx, `
INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id`,
		coupon.Code, coupon.BenefitID, coupon.BusinessID, coupon.CustomerID, req.TokenHash, coupon.ExpiresAt, req.Now,
	).Scan(&coupon.ID)
	if err != nil {
		return benefits.Issued{}, mapInsertError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return benefits.Issued{}, fmt.Errorf("postgres: commit coupon: %w", err)
	}
	return benefits.Issued{Coupon: coupon, Benefit: benefit}, nil
}

// mapInsertError turns the two unique violations the coupon insert can hit into their
// domain errors and wraps anything else.
func mapInsertError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		switch pgErr.ConstraintName {
		case codeKey:
			return benefits.ErrCodeTaken
		case onePendingKey:
			// Two requests of the same customer raced past the check above.
			return benefits.ErrPendingCoupon
		}
	}
	return fmt.Errorf("postgres: insert coupon: %w", err)
}
