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

const selectService = `SELECT id, business_id, branch_id, name, description, active, COALESCE(price, 0) FROM services`

// Services implements business.ServiceRepository. Every statement filters by
// business_id, so a service id from another business behaves as if it did not exist.
type Services struct {
	pool *pgxpool.Pool
}

// NewServices builds the repository over pool.
func NewServices(pool *pgxpool.Pool) *Services {
	return &Services{pool: pool}
}

func insertService(ctx context.Context, q rowQuerier, s domain.Service) (domain.Service, error) {
	err := q.QueryRow(ctx, `
INSERT INTO services (business_id, branch_id, name, description, active, price)
VALUES ($1, NULLIF($2, 0), $3, $4, $5, NULLIF($6, 0))
RETURNING id`,
		s.BusinessID, s.BranchID, s.Name, s.Description, s.Active, s.Price,
	).Scan(&s.ID)
	if err := mapBranchError(err); err != nil {
		return domain.Service{}, err
	}
	return s, nil
}

// Create stores s and returns it with ID set, without looking at the plan. A branch
// of another business is refused by the composite foreign key and reported as
// business.ErrBranchNotFound. The use cases go through CreateIfRoom.
func (r *Services) Create(ctx context.Context, s domain.Service) (domain.Service, error) {
	return insertService(ctx, r.pool, s)
}

// CreateIfRoom counts the active services and stores s under one lock on the
// business, so concurrent requests cannot both take the last place of the plan.
func (r *Services) CreateIfRoom(ctx context.Context, s domain.Service, fits business.Fits) (domain.Service, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Service{}, fmt.Errorf("postgres: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lock(ctx, tx, s.BusinessID); err != nil {
		return domain.Service{}, err
	}
	if s.Active {
		var active int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM services WHERE business_id = $1 AND active`, s.BusinessID).Scan(&active); err != nil {
			return domain.Service{}, fmt.Errorf("postgres: count active services: %w", err)
		}
		if !fits(active) {
			return domain.Service{}, business.ErrServiceLimitReached
		}
	}

	stored, err := insertService(ctx, tx, s)
	if err != nil {
		return domain.Service{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Service{}, fmt.Errorf("postgres: commit: %w", err)
	}
	return stored, nil
}

// SetActive flips the flag of one service. Turning it on counts the active ones
// under the lock on the business, like CreateIfRoom.
func (r *Services) SetActive(ctx context.Context, businessID, serviceID int64, active bool, fits business.Fits) (domain.Service, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Service{}, fmt.Errorf("postgres: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lock(ctx, tx, businessID); err != nil {
		return domain.Service{}, err
	}
	current, err := scanService(tx.QueryRow(ctx, selectService+` WHERE id = $1 AND business_id = $2 AND deleted_at IS NULL`, serviceID, businessID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Service{}, business.ErrServiceNotFound
	}
	if err != nil {
		return domain.Service{}, err
	}
	if current.Active == active {
		return current, nil
	}

	if active {
		var others int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM services WHERE business_id = $1 AND active`, businessID).Scan(&others); err != nil {
			return domain.Service{}, fmt.Errorf("postgres: count active services: %w", err)
		}
		if !fits(others) {
			return domain.Service{}, business.ErrServiceLimitReached
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE services SET active = $3, updated_at = now() WHERE id = $1 AND business_id = $2`, serviceID, businessID, active); err != nil {
		return domain.Service{}, fmt.Errorf("postgres: set service active: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Service{}, fmt.Errorf("postgres: commit: %w", err)
	}

	current.Active = active
	return current, nil
}

// List returns every service of the business that was not deleted, oldest first.
func (r *Services) List(ctx context.Context, businessID int64) ([]domain.Service, error) {
	rows, err := r.pool.Query(ctx, selectService+` WHERE business_id = $1 AND deleted_at IS NULL ORDER BY id`, businessID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list services: %w", err)
	}
	defer rows.Close()

	services := []domain.Service{}
	for rows.Next() {
		s, err := scanService(rows)
		if err != nil {
			return nil, err
		}
		services = append(services, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: list services: %w", err)
	}
	return services, nil
}

// Find returns one service of the business.
func (r *Services) Find(ctx context.Context, businessID, serviceID int64) (domain.Service, error) {
	s, err := scanService(r.pool.QueryRow(ctx, selectService+` WHERE id = $1 AND business_id = $2 AND deleted_at IS NULL`, serviceID, businessID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Service{}, business.ErrServiceNotFound
	}
	return s, err
}

// Save persists the details of an existing service. The active flag is left alone:
// SetActive owns it, so an edit cannot undo a concurrent on or off.
func (r *Services) Save(ctx context.Context, s domain.Service) error {
	tag, err := r.pool.Exec(ctx, `
UPDATE services
SET branch_id = NULLIF($3, 0), name = $4, description = $5, price = NULLIF($6, 0), updated_at = now()
WHERE id = $1 AND business_id = $2 AND deleted_at IS NULL`,
		s.ID, s.BusinessID, s.BranchID, s.Name, s.Description, s.Price)
	if err := mapBranchError(err); err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return business.ErrServiceNotFound
	}
	return nil
}

// Delete marks the service deleted and turns it off, so it leaves the plan count, the
// panel and the public profile. The row stays: metric_daily keeps pointing at it.
func (r *Services) Delete(ctx context.Context, businessID, serviceID int64) error {
	tag, err := r.pool.Exec(ctx, `
UPDATE services SET active = false, deleted_at = now(), updated_at = now()
WHERE id = $1 AND business_id = $2 AND deleted_at IS NULL`, serviceID, businessID)
	if err != nil {
		return fmt.Errorf("postgres: delete service: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return business.ErrServiceNotFound
	}
	return nil
}

// Schedule returns the weekly slots ordered by weekday and start.
func (r *Services) Schedule(ctx context.Context, businessID, serviceID int64) ([]domain.Slot, error) {
	if _, err := r.Find(ctx, businessID, serviceID); err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx, `
SELECT weekday, to_char(starts_at, 'HH24:MI'), to_char(ends_at, 'HH24:MI')
FROM service_schedules WHERE service_id = $1 ORDER BY weekday, starts_at`, serviceID)
	if err != nil {
		return nil, fmt.Errorf("postgres: select schedule: %w", err)
	}
	defer rows.Close()

	slots := []domain.Slot{}
	for rows.Next() {
		var (
			weekday      int16
			starts, ends string
		)
		if err := rows.Scan(&weekday, &starts, &ends); err != nil {
			return nil, fmt.Errorf("postgres: scan schedule: %w", err)
		}
		start, err := domain.ParseClock(starts)
		if err != nil {
			return nil, fmt.Errorf("postgres: stored start %q: %w", starts, err)
		}
		end, err := domain.ParseClock(ends)
		if err != nil {
			return nil, fmt.Errorf("postgres: stored end %q: %w", ends, err)
		}
		slots = append(slots, domain.Slot{Weekday: int(weekday), Start: start, End: end})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: select schedule: %w", err)
	}
	return slots, nil
}

// ReplaceSchedule swaps all the slots of the service in one transaction: either
// the new schedule is stored whole or the previous one stays.
func (r *Services) ReplaceSchedule(ctx context.Context, businessID, serviceID int64, slots []domain.Slot) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var locked int64
	err = tx.QueryRow(ctx, `SELECT id FROM services WHERE id = $1 AND business_id = $2 AND deleted_at IS NULL FOR UPDATE`, serviceID, businessID).Scan(&locked)
	if errors.Is(err, pgx.ErrNoRows) {
		return business.ErrServiceNotFound
	}
	if err != nil {
		return fmt.Errorf("postgres: lock service: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM service_schedules WHERE service_id = $1`, serviceID); err != nil {
		return fmt.Errorf("postgres: clear schedule: %w", err)
	}
	for _, s := range slots {
		if _, err := tx.Exec(ctx,
			`INSERT INTO service_schedules (service_id, weekday, starts_at, ends_at) VALUES ($1, $2, $3, $4)`,
			serviceID, s.Weekday, s.Start.String(), s.End.String()); err != nil {
			return fmt.Errorf("postgres: insert slot: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres: commit schedule: %w", err)
	}
	return nil
}

// mapBranchError turns the composite foreign key violation into
// business.ErrBranchNotFound and wraps anything else.
func mapBranchError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolation {
		return business.ErrBranchNotFound
	}
	return fmt.Errorf("postgres: write service: %w", err)
}

func scanService(row pgx.Row) (domain.Service, error) {
	var (
		s      domain.Service
		branch *int64
	)
	if err := row.Scan(&s.ID, &s.BusinessID, &branch, &s.Name, &s.Description, &s.Active, &s.Price); err != nil {
		return domain.Service{}, fmt.Errorf("postgres: scan service: %w", err)
	}
	if branch != nil {
		s.BranchID = *branch
	}
	return s, nil
}
