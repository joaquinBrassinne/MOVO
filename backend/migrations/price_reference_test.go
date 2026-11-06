package migrations_test

import (
	"strings"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

// TestServicePriceAndBranchReferenceAreOptionalAndBounded pins the two columns the
// owner panel added in phase 2: an optional price in whole pesos and a short reference.
func TestServicePriceAndBranchReferenceAreOptionalAndBounded(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	business, err := insertBusiness(ctx, pool, insertOwner(ctx, t, pool, "owner@example.com"), "mine", "5491112345678", "draft")
	if err != nil {
		t.Fatalf("insert business: %v", err)
	}

	service := func(price any) error {
		_, err := pool.Exec(ctx, `INSERT INTO services (business_id, name, price) VALUES ($1, 'Yoga', $2)`, business, price)
		return err
	}
	if err := service(nil); err != nil {
		t.Fatalf("a service without price must be allowed: %v", err)
	}
	if err := service(5500); err != nil {
		t.Fatalf("a priced service must be allowed: %v", err)
	}
	requireSQLState(t, service(-1), checkViolation)
	requireSQLState(t, service(100_000_001), checkViolation)

	branch := func(reference string) error {
		_, err := pool.Exec(ctx,
			`INSERT INTO branches (business_id, name, address, city, location, reference)
			 VALUES ($1, 'Centro', 'Calle 1', 'Rosario', ST_SetSRID(ST_MakePoint(-60.65, -32.95), 4326)::geography, $2)`,
			business, reference)
		return err
	}
	if err := branch("PB, timbre 4B"); err != nil {
		t.Fatalf("a reference must be allowed: %v", err)
	}
	requireSQLState(t, branch(strings.Repeat("a", 101)), checkViolation)

	var reference string
	if err := pool.QueryRow(ctx, `INSERT INTO branches (business_id, name, address, city, location)
		VALUES ($1, 'Norte', 'Calle 2', 'Rosario', ST_SetSRID(ST_MakePoint(-60.65, -32.95), 4326)::geography) RETURNING reference`, business).Scan(&reference); err != nil || reference != "" {
		t.Errorf("default reference = %q, %v; want empty", reference, err)
	}
}
