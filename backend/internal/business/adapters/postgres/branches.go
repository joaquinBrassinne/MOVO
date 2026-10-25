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

const selectBranch = `
SELECT id, business_id, name, address, city, ST_Y(location::geometry), ST_X(location::geometry),
       external_booking_url, active, reference
FROM branches`

// Branches implements business.BranchRepository. Every statement filters by
// business_id, so a branch id from another business behaves as if it did not exist.
type Branches struct {
	pool *pgxpool.Pool
}

// NewBranches builds the repository over pool.
func NewBranches(pool *pgxpool.Pool) *Branches {
	return &Branches{pool: pool}
}

// Create stores b and returns it with ID set.
func (r *Branches) Create(ctx context.Context, b domain.Branch) (domain.Branch, error) {
	err := r.pool.QueryRow(ctx, `
INSERT INTO branches (business_id, name, address, city, location, external_booking_url, active, reference)
VALUES ($1, $2, $3, $4, ST_SetSRID(ST_MakePoint($6, $5), 4326)::geography, NULLIF($7, ''), $8, $9)
RETURNING id`,
		b.BusinessID, b.Name, b.Address, b.City, b.Latitude, b.Longitude, b.BookingURL, b.Active, b.Reference,
	).Scan(&b.ID)
	if err != nil {
		return domain.Branch{}, fmt.Errorf("postgres: insert branch: %w", err)
	}
	return b, nil
}

// List returns the branches of the business, oldest first.
func (r *Branches) List(ctx context.Context, businessID int64) ([]domain.Branch, error) {
	rows, err := r.pool.Query(ctx, selectBranch+` WHERE business_id = $1 ORDER BY id`, businessID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list branches: %w", err)
	}
	defer rows.Close()

	branches := []domain.Branch{}
	for rows.Next() {
		b, err := scanBranch(rows)
		if err != nil {
			return nil, err
		}
		branches = append(branches, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: list branches: %w", err)
	}
	return branches, nil
}

// Find returns one branch of the business.
func (r *Branches) Find(ctx context.Context, businessID, branchID int64) (domain.Branch, error) {
	b, err := scanBranch(r.pool.QueryRow(ctx, selectBranch+` WHERE id = $1 AND business_id = $2`, branchID, businessID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Branch{}, business.ErrBranchNotFound
	}
	return b, err
}

// Save persists the details of an existing branch.
func (r *Branches) Save(ctx context.Context, b domain.Branch) error {
	tag, err := r.pool.Exec(ctx, `
UPDATE branches
SET name = $3, address = $4, city = $5, location = ST_SetSRID(ST_MakePoint($7, $6), 4326)::geography,
    external_booking_url = NULLIF($8, ''), active = $9, reference = $10, updated_at = now()
WHERE id = $1 AND business_id = $2`,
		b.ID, b.BusinessID, b.Name, b.Address, b.City, b.Latitude, b.Longitude, b.BookingURL, b.Active, b.Reference)
	if err != nil {
		return fmt.Errorf("postgres: update branch: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return business.ErrBranchNotFound
	}
	return nil
}

// Hours returns the opening slots of the branch ordered by weekday and start.
func (r *Branches) Hours(ctx context.Context, businessID, branchID int64) ([]domain.Slot, error) {
	if _, err := r.Find(ctx, businessID, branchID); err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx, `
SELECT weekday, to_char(opens_at, 'HH24:MI'), to_char(closes_at, 'HH24:MI')
FROM business_hours WHERE branch_id = $1 ORDER BY weekday, opens_at`, branchID)
	if err != nil {
		return nil, fmt.Errorf("postgres: select hours: %w", err)
	}
	defer rows.Close()

	slots := []domain.Slot{}
	for rows.Next() {
		var (
			weekday     int16
			opens, ends string
		)
		if err := rows.Scan(&weekday, &opens, &ends); err != nil {
			return nil, fmt.Errorf("postgres: scan hours: %w", err)
		}
		start, err := domain.ParseClock(opens)
		if err != nil {
			return nil, fmt.Errorf("postgres: stored opening time %q: %w", opens, err)
		}
		end, err := domain.ParseClock(ends)
		if err != nil {
			return nil, fmt.Errorf("postgres: stored closing time %q: %w", ends, err)
		}
		slots = append(slots, domain.Slot{Weekday: int(weekday), Start: start, End: end})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: select hours: %w", err)
	}
	return slots, nil
}

// ReplaceHours swaps all the slots of the branch in one transaction: either the
// new schedule is stored whole or the previous one stays.
func (r *Branches) ReplaceHours(ctx context.Context, businessID, branchID int64, slots []domain.Slot) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var locked int64
	err = tx.QueryRow(ctx, `SELECT id FROM branches WHERE id = $1 AND business_id = $2 FOR UPDATE`, branchID, businessID).Scan(&locked)
	if errors.Is(err, pgx.ErrNoRows) {
		return business.ErrBranchNotFound
	}
	if err != nil {
		return fmt.Errorf("postgres: lock branch: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM business_hours WHERE branch_id = $1`, branchID); err != nil {
		return fmt.Errorf("postgres: clear hours: %w", err)
	}
	for _, s := range slots {
		if _, err := tx.Exec(ctx,
			`INSERT INTO business_hours (branch_id, weekday, opens_at, closes_at) VALUES ($1, $2, $3, $4)`,
			branchID, s.Weekday, s.Start.String(), s.End.String()); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == exclusionViolation {
				// Two writers racing past the domain check: the database has the last word.
				return domain.ErrSlotsOverlap
			}
			return fmt.Errorf("postgres: insert slot: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres: commit hours: %w", err)
	}
	return nil
}

func scanBranch(row pgx.Row) (domain.Branch, error) {
	var (
		b       domain.Branch
		booking *string
	)
	err := row.Scan(&b.ID, &b.BusinessID, &b.Name, &b.Address, &b.City, &b.Latitude, &b.Longitude, &booking, &b.Active, &b.Reference)
	if err != nil {
		return domain.Branch{}, fmt.Errorf("postgres: scan branch: %w", err)
	}
	if booking != nil {
		b.BookingURL = *booking
	}
	return b, nil
}
