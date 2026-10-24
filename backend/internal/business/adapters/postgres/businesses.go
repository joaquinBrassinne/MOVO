// Package postgres implements the business ports on top of PostgreSQL.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

const (
	uniqueViolation     = "23505"
	foreignKeyViolation = "23503"
)

const selectBusiness = `
SELECT id, owner_user_id, category_id, name, slug, description, whatsapp_number, status,
       pilot, verified_at, verified_by, payment_reference, created_at
FROM businesses`

// Businesses implements business.BusinessRepository.
type Businesses struct {
	pool *pgxpool.Pool
}

// NewBusinesses builds the repository over pool.
func NewBusinesses(pool *pgxpool.Pool) *Businesses {
	return &Businesses{pool: pool}
}

// Create stores b and returns it with ID and CreatedAt set.
func (r *Businesses) Create(ctx context.Context, b domain.Business) (domain.Business, error) {
	err := r.pool.QueryRow(ctx, `
INSERT INTO businesses (owner_user_id, category_id, name, slug, description, whatsapp_number, status)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, created_at`,
		b.OwnerID, b.CategoryID, b.Name, b.Slug, b.Description, b.WhatsApp, string(b.Status),
	).Scan(&b.ID, &b.CreatedAt)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch {
		case pgErr.Code == uniqueViolation && pgErr.ConstraintName == "businesses_owner_user_id_key":
			return domain.Business{}, business.ErrOwnerHasBusiness
		case pgErr.Code == uniqueViolation && pgErr.ConstraintName == "businesses_slug_key":
			return domain.Business{}, business.ErrSlugTaken
		case pgErr.Code == foreignKeyViolation && pgErr.ConstraintName == "businesses_category_id_fkey":
			return domain.Business{}, business.ErrCategoryNotFound
		}
	}
	if err != nil {
		return domain.Business{}, fmt.Errorf("postgres: insert business: %w", err)
	}
	return b, nil
}

// FindByOwner returns the business of the owner.
func (r *Businesses) FindByOwner(ctx context.Context, ownerID int64) (domain.Business, error) {
	var (
		b       domain.Business
		status  string
		by      *int64
		payment *string
	)
	err := r.pool.QueryRow(ctx, selectBusiness+` WHERE owner_user_id = $1`, ownerID).Scan(
		&b.ID, &b.OwnerID, &b.CategoryID, &b.Name, &b.Slug, &b.Description, &b.WhatsApp, &status,
		&b.Pilot, &b.VerifiedAt, &by, &payment, &b.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Business{}, business.ErrBusinessNotFound
	}
	if err != nil {
		return domain.Business{}, fmt.Errorf("postgres: select business: %w", err)
	}

	// The CHECK constraint keeps the column to the known statuses.
	b.Status = domain.Status(status)
	if by != nil {
		b.VerifiedBy = *by
	}
	if payment != nil {
		b.PaymentReference = *payment
	}
	return b, nil
}

// Save persists the profile, status and payment reference of b. The slug and
// the owner are deliberately not part of the UPDATE.
func (r *Businesses) Save(ctx context.Context, b domain.Business) error {
	tag, err := r.pool.Exec(ctx, `
UPDATE businesses
SET category_id = $2, name = $3, description = $4, whatsapp_number = $5, status = $6,
    payment_reference = NULLIF($7, ''), updated_at = now()
WHERE id = $1`,
		b.ID, b.CategoryID, b.Name, b.Description, b.WhatsApp, string(b.Status), b.PaymentReference)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolation && pgErr.ConstraintName == "businesses_category_id_fkey" {
		return business.ErrCategoryNotFound
	}
	if err != nil {
		return fmt.Errorf("postgres: update business: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return business.ErrBusinessNotFound
	}
	return nil
}

// IsPilot reports whether the business is part of the pilot. It is not part of
// business.BusinessRepository: the billing module declares the port that needs it.
func (r *Businesses) IsPilot(ctx context.Context, businessID int64) (bool, error) {
	var pilot bool
	err := r.pool.QueryRow(ctx, `SELECT pilot FROM businesses WHERE id = $1`, businessID).Scan(&pilot)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, business.ErrBusinessNotFound
	}
	if err != nil {
		return false, fmt.Errorf("postgres: select pilot flag: %w", err)
	}
	return pilot, nil
}
