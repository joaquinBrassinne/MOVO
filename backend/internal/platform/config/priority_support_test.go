package config_test

import (
	"strings"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
)

func TestLoadPlansPrioritySupportIsOptionalAndDefaultsToProOnly(t *testing.T) {
	t.Parallel()

	got, err := config.LoadPlans(plansEnv(nil))
	if err != nil || got.Starter.PrioritySupport || !got.Pro.PrioritySupport {
		t.Errorf("defaults: %+v, %v; want starter without and pro with the support", got, err)
	}
	got, err = config.LoadPlans(plansEnv(map[string]string{"PLAN_STARTER_PRIORITY_SUPPORT": "true"}))
	if err != nil || !got.Starter.PrioritySupport {
		t.Errorf("starter with the support: %+v, %v; want it on", got.Starter, err)
	}
	got, err = config.LoadPlans(plansEnv(map[string]string{"PLAN_PRO_PRIORITY_SUPPORT": "false"}))
	if err != nil || got.Pro.PrioritySupport {
		t.Errorf("pro without the support: %+v, %v; want it off", got.Pro, err)
	}
}

func TestLoadPlansPrioritySupportRejections(t *testing.T) {
	t.Parallel()

	for name, overrides := range map[string]map[string]string{
		"a typo is not a bool":               {"PLAN_PRO_PRIORITY_SUPPORT": "sure"},
		"pro cannot offer less than starter": {"PLAN_STARTER_PRIORITY_SUPPORT": "true", "PLAN_PRO_PRIORITY_SUPPORT": "false"},
	} {
		if _, err := config.LoadPlans(plansEnv(overrides)); err == nil || !strings.Contains(err.Error(), "PRIORITY_SUPPORT") {
			t.Errorf("%s: error = %v, want one naming PRIORITY_SUPPORT", name, err)
		}
	}
}
