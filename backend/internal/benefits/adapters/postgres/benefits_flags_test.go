package postgres_test

import (
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

// TestSaveAndSetActiveOwnTheirOwnColumns pins that an edit holding a stale copy cannot
// undo a concurrent on or off, and that flipping the switch never rewrites the details.
func TestSaveAndSetActiveOwnTheirOwnColumns(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newShop(ctx, t, pool, "studio", "active", 0)
	repo := postgres.NewBenefits(pool)

	stale, err := repo.Find(ctx, s.business, s.benefit)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if !stale.Active {
		t.Fatal("the fixture benefit should start active")
	}

	off, err := repo.SetActive(ctx, s.business, s.benefit, false)
	if err != nil || off.Active {
		t.Fatalf("SetActive(false) = %+v, %v; want an inactive benefit", off, err)
	}

	// An edit that read the benefit before it was switched off saves its details only.
	stale.Percent = 25
	if err := repo.Save(ctx, stale); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := repo.Find(ctx, s.business, s.benefit)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if got.Active {
		t.Error("Save switched the benefit back on")
	}
	if got.Percent != 25 {
		t.Errorf("percent = %d, want 25 saved", got.Percent)
	}

	on, err := repo.SetActive(ctx, s.business, s.benefit, true)
	if err != nil || !on.Active || on.Percent != 25 {
		t.Fatalf("SetActive(true) = %+v, %v; want active with the details kept", on, err)
	}

	if _, err := repo.SetActive(ctx, s.business+999, s.benefit, false); !errors.Is(err, benefits.ErrBenefitNotFound) {
		t.Errorf("SetActive through another business error = %v, want ErrBenefitNotFound", err)
	}
}
