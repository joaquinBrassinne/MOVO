package domain_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/billing/domain"
)

func mustEntitlements(t *testing.T, branches, services int) domain.Entitlements {
	t.Helper()

	e, err := domain.NewEntitlements(branches, services)
	if err != nil {
		t.Fatalf("NewEntitlements(%d, %d): %v", branches, services, err)
	}
	return e
}

func TestParsePlan(t *testing.T) {
	t.Parallel()

	for _, plan := range []domain.Plan{domain.PlanStarter, domain.PlanPro} {
		got, err := domain.ParsePlan(string(plan))
		if err != nil || got != plan {
			t.Errorf("ParsePlan(%q) = %q, %v", plan, got, err)
		}
	}
	for _, raw := range []string{"", "free", "Pro", "enterprise"} {
		if _, err := domain.ParsePlan(raw); !errors.Is(err, domain.ErrPlanInvalid) {
			t.Errorf("ParsePlan(%q) error = %v, want ErrPlanInvalid", raw, err)
		}
	}
}

func TestNewEntitlementsRejectsNegativeLimits(t *testing.T) {
	t.Parallel()

	if _, err := domain.NewEntitlements(-1, 3); !errors.Is(err, domain.ErrLimitInvalid) {
		t.Errorf("negative branches error = %v, want ErrLimitInvalid", err)
	}
	if _, err := domain.NewEntitlements(2, -1); !errors.Is(err, domain.ErrLimitInvalid) {
		t.Errorf("negative services error = %v, want ErrLimitInvalid", err)
	}
}

func TestEntitlementsLimitsAreCheckedAtCreationOrActivation(t *testing.T) {
	t.Parallel()

	starter := mustEntitlements(t, 2, 3)
	tests := []struct {
		name string
		got  bool
		want bool
	}{
		{"first branch", starter.CanAddBranch(0), true},
		{"second branch", starter.CanAddBranch(1), true},
		{"third branch over the limit", starter.CanAddBranch(2), false},
		{"far over the limit after a downgrade", starter.CanAddBranch(5), false},
		{"third service", starter.CanActivateService(2), true},
		{"fourth service over the limit", starter.CanActivateService(3), false},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}

	pro := mustEntitlements(t, 4, 0)
	if !pro.CanActivateService(1000) {
		t.Error("0 services means unlimited, so activation must always be allowed")
	}
	if pro.CanAddBranch(4) {
		t.Error("the fifth branch must not fit in 4")
	}
}

func TestDowngradeKeepsTheOldestAndDeactivatesTheRest(t *testing.T) {
	t.Parallel()

	starter := mustEntitlements(t, 2, 3)

	// Ids are given oldest first, the order the repositories return them in.
	if got, want := starter.BranchesToDeactivate([]int64{10, 11, 12, 13}), []int64{12, 13}; !slices.Equal(got, want) {
		t.Errorf("branches to deactivate = %v, want %v", got, want)
	}
	if got := starter.BranchesToDeactivate([]int64{10, 11}); len(got) != 0 {
		t.Errorf("within the limit nothing is deactivated, got %v", got)
	}
	if got, want := starter.ServicesToDeactivate([]int64{1, 2, 3, 4, 5}), []int64{4, 5}; !slices.Equal(got, want) {
		t.Errorf("services to deactivate = %v, want %v", got, want)
	}

	unlimited := mustEntitlements(t, 4, 0)
	if got := unlimited.ServicesToDeactivate([]int64{1, 2, 3, 4, 5, 6}); len(got) != 0 {
		t.Errorf("unlimited services must keep everything, got %v", got)
	}
}

func TestDowngradeDoesNotMutateItsInput(t *testing.T) {
	t.Parallel()

	ids := []int64{1, 2, 3, 4}
	_ = mustEntitlements(t, 2, 2).BranchesToDeactivate(ids)
	if !slices.Equal(ids, []int64{1, 2, 3, 4}) {
		t.Errorf("input = %v, must be left untouched", ids)
	}
}
