package queries_test

import (
	"context"
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/queries"
)

type fakePermission struct {
	allowed bool
	err     error
	asked   []int64
}

func (f *fakePermission) AdvancedMetrics(_ context.Context, businessID int64) (bool, error) {
	f.asked = append(f.asked, businessID)
	return f.allowed, f.err
}

func TestProGateLetsAPlanWithAdvancedMetricsThrough(t *testing.T) {
	t.Parallel()

	permission := &fakePermission{allowed: true}
	id, err := queries.NewProGate(fakeOwners{business: 42}, permission).Business(t.Context(), 7)
	if err != nil || id != 42 {
		t.Fatalf("Business = %d, %v; want 42 and no error", id, err)
	}
	if len(permission.asked) != 1 || permission.asked[0] != 42 {
		t.Errorf("permission asked for %v, want the owner's business 42", permission.asked)
	}
}

func TestProGateRefusesAPlanWithoutAdvancedMetrics(t *testing.T) {
	t.Parallel()

	id, err := queries.NewProGate(fakeOwners{business: 42}, &fakePermission{allowed: false}).Business(t.Context(), 7)
	if !errors.Is(err, benefits.ErrPlanUpgradeRequired) || id != 0 {
		t.Errorf("Business = %d, %v; want 0 and ErrPlanUpgradeRequired", id, err)
	}
}

func TestProGateStopsWhereTheOwnerHasNoBusinessAndNeverGuessesOnAFailure(t *testing.T) {
	t.Parallel()

	permission := &fakePermission{allowed: true}
	if _, err := queries.NewProGate(fakeOwners{err: benefits.ErrBusinessNotFound}, permission).Business(t.Context(), 7); !errors.Is(err, benefits.ErrBusinessNotFound) {
		t.Errorf("no business: err = %v, want ErrBusinessNotFound", err)
	}
	if len(permission.asked) != 0 {
		t.Errorf("permission was asked %d times without a business, want none", len(permission.asked))
	}

	boom := errors.New("database down")
	_, err := queries.NewProGate(fakeOwners{business: 1}, &fakePermission{err: boom, allowed: true}).Business(t.Context(), 7)
	if !errors.Is(err, boom) || errors.Is(err, benefits.ErrPlanUpgradeRequired) {
		t.Errorf("failed lookup: err = %v, want the lookup's error and not an upgrade request", err)
	}
}
