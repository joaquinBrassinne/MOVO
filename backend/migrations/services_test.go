package migrations_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

func insertBranch(ctx context.Context, t *testing.T, pool *pgxpool.Pool, businessID int64) int64 {
	t.Helper()

	var id int64
	err := pool.QueryRow(ctx,
		`INSERT INTO branches (business_id, name, address, city, location)
		 VALUES ($1, 'Centro', 'Calle 1', 'Rosario', ST_SetSRID(ST_MakePoint(-60.65, -32.95), 4326)::geography) RETURNING id`,
		businessID).Scan(&id)
	if err != nil {
		t.Fatalf("insert branch: %v", err)
	}
	return id
}

// TestServicesBelongToTheirBusinessAndBranch pins that a service may be tied to a
// branch only of its own business, and that the branch is optional.
func TestServicesBelongToTheirBusinessAndBranch(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	mine, err := insertBusiness(ctx, pool, insertOwner(ctx, t, pool, "owner@example.com"), "mine", "5491112345678", "draft")
	if err != nil {
		t.Fatalf("insert business: %v", err)
	}
	theirs, err := insertBusiness(ctx, pool, insertOwner(ctx, t, pool, "other@example.com"), "theirs", "5491112345678", "draft")
	if err != nil {
		t.Fatalf("insert business: %v", err)
	}
	myBranch := insertBranch(ctx, t, pool, mine)

	insert := func(businessID int64, branchID any, name string) error {
		_, err := pool.Exec(ctx, `INSERT INTO services (business_id, branch_id, name) VALUES ($1, $2, $3)`, businessID, branchID, name)
		return err
	}

	if err := insert(mine, nil, "Yoga"); err != nil {
		t.Fatalf("a service without branch must be allowed: %v", err)
	}
	if err := insert(mine, myBranch, "Pilates"); err != nil {
		t.Fatalf("a service in its own branch must be allowed: %v", err)
	}
	requireSQLState(t, insert(theirs, myBranch, "Hijack"), foreignKeyViolation)
	requireSQLState(t, insert(mine, nil, "  "), checkViolation)

	var active bool
	if err := pool.QueryRow(ctx, `SELECT active FROM services WHERE name = 'Yoga'`).Scan(&active); err != nil || !active {
		t.Errorf("default active = %v, %v; want true", active, err)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM branches WHERE id = $1`, myBranch); err != nil {
		t.Fatalf("delete branch: %v", err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM services WHERE name = 'Pilates'`).Scan(&n); err != nil || n != 0 {
		t.Errorf("services of a deleted branch = %d, %v; want 0", n, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM services WHERE name = 'Yoga'`).Scan(&n); err != nil || n != 1 {
		t.Errorf("service without branch = %d, %v; want it untouched", n, err)
	}
}

// TestServiceSchedulesEnforceTheirRules pins weekday range, ordered intervals,
// split shifts and the cascade from the service.
func TestServiceSchedulesEnforceTheirRules(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	business, err := insertBusiness(ctx, pool, insertOwner(ctx, t, pool, "owner@example.com"), "mine", "5491112345678", "draft")
	if err != nil {
		t.Fatalf("insert business: %v", err)
	}
	var service int64
	if err := pool.QueryRow(ctx, `INSERT INTO services (business_id, name) VALUES ($1, 'Yoga') RETURNING id`, business).Scan(&service); err != nil {
		t.Fatalf("insert service: %v", err)
	}

	slot := func(weekday int, starts, ends string) error {
		_, err := pool.Exec(ctx,
			`INSERT INTO service_schedules (service_id, weekday, starts_at, ends_at) VALUES ($1, $2, $3, $4)`,
			service, weekday, starts, ends)
		return err
	}
	if err := slot(1, "09:00", "10:00"); err != nil {
		t.Fatalf("insert slot: %v", err)
	}
	if err := slot(1, "18:00", "19:00"); err != nil {
		t.Fatalf("two classes on the same day must be allowed: %v", err)
	}
	requireSQLState(t, slot(7, "09:00", "10:00"), checkViolation)
	requireSQLState(t, slot(2, "10:00", "09:00"), checkViolation)
	requireSQLState(t, slot(1, "09:00", "11:00"), uniqueViolation)

	if _, err := pool.Exec(ctx, `DELETE FROM services WHERE id = $1`, service); err != nil {
		t.Fatalf("delete service: %v", err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM service_schedules`).Scan(&n); err != nil || n != 0 {
		t.Errorf("schedules left after deleting the service = %d, %v; want 0", n, err)
	}
}
