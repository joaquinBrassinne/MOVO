package pilotplan_test

import (
	"context"
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/billing/adapters/fixedplan"
	"github.com/joaquinBrassinne/MOVO/internal/billing/adapters/pilotplan"
	"github.com/joaquinBrassinne/MOVO/internal/billing/domain"
)

type pilots struct {
	ids map[int64]bool
	err error
}

func (p pilots) IsPilot(_ context.Context, businessID int64) (bool, error) {
	return p.ids[businessID], p.err
}

func TestPilotsGetProAndEveryoneElseFallsThrough(t *testing.T) {
	t.Parallel()

	resolver := pilotplan.New(pilots{ids: map[int64]bool{1: true}}, fixedplan.New(domain.PlanStarter))

	if got, err := resolver.PlanOf(t.Context(), 1); err != nil || got != domain.PlanPro {
		t.Errorf("pilot plan = %q, %v; want pro", got, err)
	}
	if got, err := resolver.PlanOf(t.Context(), 2); err != nil || got != domain.PlanStarter {
		t.Errorf("regular business plan = %q, %v; want the fallback (starter)", got, err)
	}
}

func TestPilotLookupFailureIsNotSwallowed(t *testing.T) {
	t.Parallel()

	boom := errors.New("database down")
	resolver := pilotplan.New(pilots{err: boom}, fixedplan.New(domain.PlanStarter))

	if _, err := resolver.PlanOf(t.Context(), 1); !errors.Is(err, boom) {
		t.Fatalf("error = %v, want the lookup error: guessing a plan would grant or deny limits wrongly", err)
	}
}
