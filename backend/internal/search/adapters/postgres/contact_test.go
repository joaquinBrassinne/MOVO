package postgres_test

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
	"github.com/joaquinBrassinne/MOVO/internal/search"
	"github.com/joaquinBrassinne/MOVO/internal/search/adapters/postgres"
)

func addService(t *testing.T, pool *pgxpool.Pool, business int64, branch any, name string, active bool) int64 {
	t.Helper()

	var id int64
	if err := pool.QueryRow(t.Context(),
		`INSERT INTO services (business_id, branch_id, name, active) VALUES ($1, $2, $3, $4) RETURNING id`,
		business, branch, name, active).Scan(&id); err != nil {
		t.Fatalf("insert service %s: %v", name, err)
	}
	return id
}

func businessOf(t *testing.T, pool *pgxpool.Pool, branch int64) int64 {
	t.Helper()

	var id int64
	if err := pool.QueryRow(t.Context(), `SELECT business_id FROM branches WHERE id = $1`, branch).Scan(&id); err != nil {
		t.Fatalf("business of branch: %v", err)
	}
	return id
}

func TestContactForReturnsTheNumberOfAVisibleBranch(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	s := &seeded{pool: pool}
	branch := s.add(t, "yoga", "active", 100, false)

	c, err := postgres.NewBranches(pool).ContactFor(t.Context(), branch, 0)
	if err != nil {
		t.Fatalf("ContactFor: %v", err)
	}
	if c.BranchID != branch || c.WhatsApp != "5491112345678" || c.BusinessName != "Studio 1" || c.ServiceID != 0 {
		t.Errorf("contact = %+v", c)
	}
}

func TestContactForHidesBranchesThatAreNotPublic(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	s := &seeded{pool: pool}
	paused := s.add(t, "yoga", "paused", 100, false)
	inactive := s.add(t, "yoga", "active", 100, false)
	if _, err := pool.Exec(t.Context(), `UPDATE branches SET active = false WHERE id = $1`, inactive); err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	repo := postgres.NewBranches(pool)

	for name, id := range map[string]int64{"paused business": paused, "inactive branch": inactive, "unknown branch": 999_999} {
		if _, err := repo.ContactFor(t.Context(), id, 0); !errors.Is(err, search.ErrBranchNotFound) {
			t.Errorf("%s: err = %v, want ErrBranchNotFound", name, err)
		}
	}
}

// TestContactForOnlyKeepsAServiceThatQualifies pins that a service rides along only
// when it is active and belongs to this business and, if it is tied to one, this
// branch. Anything else yields the plain contact instead of an error.
func TestContactForOnlyKeepsAServiceThatQualifies(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	s := &seeded{pool: pool}
	branch := s.add(t, "yoga", "active", 100, false)
	business := businessOf(t, pool, branch)
	otherBranch := s.add(t, "padel", "active", 200, false)
	otherBusiness := businessOf(t, pool, otherBranch)

	// A second branch of the same business, to tie a service to it.
	var sibling int64
	if err := pool.QueryRow(t.Context(),
		`INSERT INTO branches (business_id, name, address, city, location)
		 VALUES ($1, 'Norte', 'Calle 2', 'Rio Tercero', ST_SetSRID(ST_MakePoint(-64.11, -32.16), 4326)::geography) RETURNING id`,
		business).Scan(&sibling); err != nil {
		t.Fatalf("insert sibling branch: %v", err)
	}

	anyBranch := addService(t, pool, business, nil, "Yoga", true)
	thisBranch := addService(t, pool, business, branch, "Pilates", true)
	siblingOnly := addService(t, pool, business, sibling, "Spinning", true)
	inactive := addService(t, pool, business, nil, "Zumba", false)
	foreign := addService(t, pool, otherBusiness, nil, "Padel", true)
	repo := postgres.NewBranches(pool)

	for name, tc := range map[string]struct {
		service  int64
		wantID   int64
		wantName string
	}{
		"business-wide service":          {anyBranch, anyBranch, "Yoga"},
		"service tied to this branch":    {thisBranch, thisBranch, "Pilates"},
		"service tied to another branch": {siblingOnly, 0, ""},
		"inactive service":               {inactive, 0, ""},
		"service of another business":    {foreign, 0, ""},
		"unknown service":                {999_999, 0, ""},
	} {
		c, err := repo.ContactFor(t.Context(), branch, tc.service)
		if err != nil {
			t.Fatalf("%s: ContactFor: %v", name, err)
		}
		if c.ServiceID != tc.wantID || c.ServiceName != tc.wantName {
			t.Errorf("%s: service = %d %q, want %d %q", name, c.ServiceID, c.ServiceName, tc.wantID, tc.wantName)
		}
	}
}
