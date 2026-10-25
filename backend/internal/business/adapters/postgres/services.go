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

// Create stores s and returns it with ID set. A branch of another business is
// refused by the composite foreign key and reported as business.ErrBranchNotFound.
func (r *Services) Create(ctx context.Context, s domain.Service) (domain.Service, error) {
	err := r.pool.QueryRow(ctx, `
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

// List returns every service of the business, oldest first.
func (r *Services) List(ctx context.Context, businessID int64) ([]domain.Service, error) {
	rows, err := r.pool.Query(ctx, selectService+` WHERE business_id = $1 ORDER BY id`, businessID)
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
	s, err := scanService(r.pool.QueryRow(ctx, selectService+` WHERE id = $1 AND business_id = $2`, serviceID, businessID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Service{}, business.ErrServiceNotFound
	}
	return s, err
}

// Save persists the details and the active flag of an existing service.
func (r *Services) Save(ctx context.Context, s domain.Service) error {
	tag, err := r.pool.Exec(ctx, `
UPDATE services
SET branch_id = NULLIF($3, 0), name = $4, description = $5, active = $6, price = NULLIF($7, 0), updated_at = now()
WHERE id = $1 AND business_id = $2`,
		s.ID, s.BusinessID, s.BranchID, s.Name, s.Description, s.Active, s.Price)
	if err := mapBranchError(err); err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return business.ErrServiceNotFound
	}
	return nil
}

// Delete removes the service; its schedule goes with it through ON DELETE CASCADE.
func (r *Services) Delete(ctx context.Context, businessID, serviceID int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM services WHERE id = $1 AND business_id = $2`, serviceID, businessID)
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
	err = tx.QueryRow(ctx, `SELECT id FROM services WHERE id = $1 AND business_id = $2 FOR UPDATE`, serviceID, businessID).Scan(&locked)
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
