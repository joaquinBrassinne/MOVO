// Package postgres implements the benefits ports on top of PostgreSQL.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

// selectBenefit reads a benefit from benefits, without an alias.
var selectBenefit = `
SELECT ` + benefitColumns("") + `
FROM benefits`

// Benefits implements benefits.BenefitRepository. Every statement filters by
// business_id, so a benefit id from another business behaves as if it did not exist.
type Benefits struct {
	pool *pgxpool.Pool
}

// NewBenefits builds the repository over pool.
func NewBenefits(pool *pgxpool.Pool) *Benefits {
	return &Benefits{pool: pool}
}

// Create stores b and returns it with ID set. A new benefit is never admin-paused.
func (r *Benefits) Create(ctx context.Context, b domain.Benefit) (domain.Benefit, error) {
	err := r.pool.QueryRow(ctx, `
INSERT INTO benefits (business_id, type, percent, amount_ars, title, conditions, active, monthly_limit)
VALUES ($1, $2, NULLIF($3, 0), NULLIF($4, 0), NULLIF($5, ''), NULLIF($6, ''), $7, $8)
RETURNING id`,
		b.BusinessID, b.Type, b.Percent, b.AmountARS, b.Title, b.Conditions, b.Active, b.MonthlyLimit,
	).Scan(&b.ID)
	if err != nil {
		return domain.Benefit{}, fmt.Errorf("postgres: create benefit: %w", err)
	}
	return b, nil
}

// List returns every benefit of the business, oldest first.
func (r *Benefits) List(ctx context.Context, businessID int64) ([]domain.Benefit, error) {
	rows, err := r.pool.Query(ctx, selectBenefit+` WHERE business_id = $1 ORDER BY id`, businessID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list benefits: %w", err)
	}
	defer rows.Close()

	list := []domain.Benefit{}
	for rows.Next() {
		b, err := scanBenefit(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: list benefits: %w", err)
	}
	return list, nil
}

// Find returns one benefit of the business.
func (r *Benefits) Find(ctx context.Context, businessID, benefitID int64) (domain.Benefit, error) {
	b, err := scanBenefit(r.pool.QueryRow(ctx, selectBenefit+` WHERE id = $1 AND business_id = $2`, benefitID, businessID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Benefit{}, benefits.ErrBenefitNotFound
	}
	return b, err
}

// Save persists the details and the owner's switch of an existing benefit. It never
// writes admin_paused: that belongs to the super admin, and an owner saving a stale copy
// must not lift a pause that happened in between.
func (r *Benefits) Save(ctx context.Context, b domain.Benefit) error {
	tag, err := r.pool.Exec(ctx, `
UPDATE benefits
SET type = $3, percent = NULLIF($4, 0), amount_ars = NULLIF($5, 0), title = NULLIF($6, ''), conditions = NULLIF($7, ''),
    active = $8, monthly_limit = $9, updated_at = now()
WHERE id = $1 AND business_id = $2`,
		b.ID, b.BusinessID, b.Type, b.Percent, b.AmountARS, b.Title, b.Conditions, b.Active, b.MonthlyLimit)
	if err != nil {
		return fmt.Errorf("postgres: save benefit: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return benefits.ErrBenefitNotFound
	}
	return nil
}

func scanBenefit(row pgx.Row) (domain.Benefit, error) {
	var b domain.Benefit
	if err := row.Scan(benefitDest(&b)...); err != nil {
		return domain.Benefit{}, fmt.Errorf("postgres: scan benefit: %w", err)
	}
	return b, nil
}
