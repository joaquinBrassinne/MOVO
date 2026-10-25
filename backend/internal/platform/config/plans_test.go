package config_test

import (
	"strings"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
)

func plansEnv(overrides map[string]string) func(string) string {
	env := map[string]string{
		"PLAN_STARTER_MAX_BRANCHES":        "2",
		"PLAN_STARTER_MAX_ACTIVE_SERVICES": "3",
		"PLAN_STARTER_PRICE":               "19800",
		"PLAN_PRO_MAX_BRANCHES":            "4",
		"PLAN_PRO_MAX_ACTIVE_SERVICES":     "0",
		"PLAN_PRO_PRICE":                   "29800",
	}
	for k, v := range overrides {
		env[k] = v
	}
	return func(name string) string { return env[name] }
}

func TestLoadPlans(t *testing.T) {
	t.Parallel()

	got, err := config.LoadPlans(plansEnv(nil))
	if err != nil {
		t.Fatalf("LoadPlans: %v", err)
	}
	want := config.Plans{
		Starter: config.PlanSettings{MaxBranches: 2, MaxActiveServices: 3, Price: 19800},
		Pro:     config.PlanSettings{MaxBranches: 4, MaxActiveServices: 0, Price: 29800},
	}
	if got != want {
		t.Errorf("plans = %+v, want %+v", got, want)
	}
}

func TestLoadPlansRejections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		overrides map[string]string
		wantInMsg string
	}{
		{"missing starter branches", map[string]string{"PLAN_STARTER_MAX_BRANCHES": ""}, "PLAN_STARTER_MAX_BRANCHES"},
		{"missing pro price", map[string]string{"PLAN_PRO_PRICE": ""}, "PLAN_PRO_PRICE"},
		{"not a number", map[string]string{"PLAN_STARTER_MAX_ACTIVE_SERVICES": "three"}, "PLAN_STARTER_MAX_ACTIVE_SERVICES"},
		{"negative limit", map[string]string{"PLAN_PRO_MAX_BRANCHES": "-1"}, "PLAN_PRO_MAX_BRANCHES"},
		{"zero branches would mean none or unlimited", map[string]string{"PLAN_STARTER_MAX_BRANCHES": "0"}, "PLAN_STARTER_MAX_BRANCHES"},
		{"zero price", map[string]string{"PLAN_STARTER_PRICE": "0"}, "PLAN_STARTER_PRICE"},
		{"pro with fewer branches than starter", map[string]string{"PLAN_PRO_MAX_BRANCHES": "1"}, "PLAN_PRO_MAX_BRANCHES"},
		{"pro with fewer services than starter", map[string]string{"PLAN_PRO_MAX_ACTIVE_SERVICES": "2"}, "PLAN_PRO_MAX_ACTIVE_SERVICES"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := config.LoadPlans(plansEnv(tt.overrides))
			if err == nil || !strings.Contains(err.Error(), tt.wantInMsg) {
				t.Fatalf("error = %v, want one naming %s", err, tt.wantInMsg)
			}
		})
	}

	if _, err := config.LoadPlans(nil); err == nil {
		t.Error("a nil lookup must be rejected")
	}
}

func TestLoadPlansAllowsUnlimitedPro(t *testing.T) {
	t.Parallel()

	got, err := config.LoadPlans(plansEnv(map[string]string{"PLAN_PRO_MAX_ACTIVE_SERVICES": "0", "PLAN_STARTER_MAX_ACTIVE_SERVICES": "0"}))
	if err != nil {
		t.Fatalf("both unlimited must be valid: %v", err)
	}
	if got.Pro.MaxActiveServices != 0 {
		t.Errorf("pro services = %d, want 0 (unlimited)", got.Pro.MaxActiveServices)
	}
}
