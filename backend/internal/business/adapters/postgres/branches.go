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

// rowQuerier is what both the pool and a transaction offer to run one statement.
type rowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func insertBranch(ctx context.Context, q rowQuerier, b domain.Branch) (domain.Branch, error) {
	err := q.QueryRow(ctx, `
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

// Create stores b and returns it with ID set, without looking at the plan. The use
// cases go through CreateIfRoom.
func (r *Branches) Create(ctx context.Context, b domain.Branch) (domain.Branch, error) {
	return insertBranch(ctx, r.pool, b)
}

// CreateIfRoom counts the active branches and stores b under one lock on the
// business, so concurrent requests cannot both take the last place of the plan.
func (r *Branches) CreateIfRoom(ctx context.Context, b domain.Branch, fits business.Fits) (domain.Branch, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Branch{}, fmt.Errorf("postgres: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lock(ctx, tx, b.BusinessID); err != nil {
		return domain.Branch{}, err
	}
	if b.Active {
		var active int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM branches WHERE business_id = $1 AND active`, b.BusinessID).Scan(&active); err != nil {
			return domain.Branch{}, fmt.Errorf("postgres: count active branches: %w", err)
		}
		if !fits(active) {
			return domain.Branch{}, business.ErrBranchLimitReached
		}
	}

	stored, err := insertBranch(ctx, tx, b)
	if err != nil {
		return domain.Branch{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Branch{}, fmt.Errorf("postgres: commit: %w", err)
	}
	return stored, nil
}

// SetActive flips the flag of one branch. Showing a hidden branch counts the active
// ones under the lock on the business, like CreateIfRoom.
func (r *Branches) SetActive(ctx context.Context, businessID, branchID int64, active bool, fits business.Fits) (domain.Branch, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Branch{}, fmt.Errorf("postgres: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lock(ctx, tx, businessID); err != nil {
		return domain.Branch{}, err
	}
	current, err := scanBranch(tx.QueryRow(ctx, selectBranch+` WHERE id = $1 AND business_id = $2`, branchID, businessID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Branch{}, business.ErrBranchNotFound
	}
	if err != nil {
		return domain.Branch{}, err
	}
	if current.Active == active {
		return current, nil
	}

	if active {
		var others int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM branches WHERE business_id = $1 AND active`, businessID).Scan(&others); err != nil {
			return domain.Branch{}, fmt.Errorf("postgres: count active branches: %w", err)
		}
		if !fits(others) {
			return domain.Branch{}, business.ErrBranchLimitReached
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE branches SET active = $3, updated_at = now() WHERE id = $1 AND business_id = $2`, branchID, businessID, active); err != nil {
		return domain.Branch{}, fmt.Errorf("postgres: set branch active: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Branch{}, fmt.Errorf("postgres: commit: %w", err)
	}

	current.Active = active
	return current, nil
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

// Save persists the details of an existing branch. The active flag is left alone:
// SetActive owns it, so an edit cannot undo a concurrent show or hide.
func (r *Branches) Save(ctx context.Context, b domain.Branch) error {
	tag, err := r.pool.Exec(ctx, `
UPDATE branches
SET name = $3, address = $4, city = $5, location = ST_SetSRID(ST_MakePoint($7, $6), 4326)::geography,
    external_booking_url = NULLIF($8, ''), reference = $9, updated_at = now()
WHERE id = $1 AND business_id = $2`,
		b.ID, b.BusinessID, b.Name, b.Address, b.City, b.Latitude, b.Longitude, b.BookingURL, b.Reference)
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
