package domain_test

import "testing"

func TestPrioritySupportIsOffUntilItIsSwitchedOnAndKeepsTheRest(t *testing.T) {
	t.Parallel()

	base := mustEntitlements(t, 4, 0).WithAdvancedMetrics(true)
	if base.PrioritySupport() {
		t.Error("a new entitlement must not include the priority support")
	}
	pro := base.WithPrioritySupport(true)
	if !pro.PrioritySupport() || !pro.AdvancedMetrics() || pro.MaxBranches() != 4 {
		t.Errorf("with the support on = %+v, want it on and the rest untouched", pro)
	}
	if base.PrioritySupport() || pro.WithPrioritySupport(false).PrioritySupport() {
		t.Error("the original must stay off and switching off must turn it off")
	}
}
