package migrations_test

import (
	"strings"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

// TestBusinessPayerColumnsAreOptionalAndBounded pins the columns that replaced the
// payment reference: what the owner declares about the Mercado Pago payer, plus the
// return URL hints, which must be a JSON object.
func TestBusinessPayerColumnsAreOptionalAndBounded(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	id, err := insertBusiness(ctx, pool, insertOwner(ctx, t, pool, "owner@example.com"), "mine", "5491112345678", "draft")
	if err != nil {
		t.Fatalf("insert business: %v", err)
	}

	var email, name, params any
	if err := pool.QueryRow(ctx, `SELECT mp_payer_email, mp_payer_name, mp_return_params FROM businesses WHERE id = $1`, id).Scan(&email, &name, &params); err != nil {
		t.Fatalf("select: %v", err)
	}
	if email != nil || name != nil || params != nil {
		t.Errorf("a new business has payer %v/%v and params %v; want none", email, name, params)
	}

	update := func(column string, value any) error {
		_, err := pool.Exec(ctx, `UPDATE businesses SET `+column+` = $2 WHERE id = $1`, id, value)
		return err
	}
	if err := update("mp_payer_email", "ana@example.com"); err != nil {
		t.Fatalf("a payer email must be allowed: %v", err)
	}
	requireSQLState(t, update("mp_payer_email", strings.Repeat("a", 255)), checkViolation)
	requireSQLState(t, update("mp_payer_name", strings.Repeat("a", 101)), checkViolation)
	if err := update("mp_return_params", `{"plan":"pro"}`); err != nil {
		t.Fatalf("an object of params must be allowed: %v", err)
	}
	requireSQLState(t, update("mp_return_params", `["plan"]`), checkViolation)

	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'businesses' AND column_name = 'payment_reference')`).Scan(&exists); err != nil || exists {
		t.Errorf("payment_reference still exists = %v, %v; want it dropped", exists, err)
	}
}
