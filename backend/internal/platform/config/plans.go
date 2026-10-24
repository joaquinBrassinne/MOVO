package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Environment variables understood by LoadPlans (SDD §7.5).
const (
	envStarterMaxBranches = "PLAN_STARTER_MAX_BRANCHES"
	envStarterMaxServices = "PLAN_STARTER_MAX_ACTIVE_SERVICES"
	envStarterPrice       = "PLAN_STARTER_PRICE"
	envProMaxBranches     = "PLAN_PRO_MAX_BRANCHES"
	envProMaxServices     = "PLAN_PRO_MAX_ACTIVE_SERVICES"
	envProPrice           = "PLAN_PRO_PRICE"
)

// PlanSettings are the limits and the display price of one plan.
type PlanSettings struct {
	// MaxBranches is at least 1.
	MaxBranches int
	// MaxActiveServices is 0 for unlimited.
	MaxActiveServices int
	// Price is the monthly price shown to owners, in whole pesos.
	Price int
}

// Plans holds the settings of both plans. Every API instance must run with the
// same values; they are validated at startup and never reloaded.
type Plans struct {
	Starter PlanSettings
	Pro     PlanSettings
}

// LoadPlans builds Plans from get, normally os.Getenv. Every variable is
// required: a missing limit would silently turn into "unlimited" or "none".
func LoadPlans(get func(string) string) (Plans, error) {
	if get == nil {
		return Plans{}, errors.New("config: environment lookup must not be nil")
	}

	var (
		plans Plans
		err   error
	)
	if plans.Starter.MaxBranches, err = requiredInt(get, envStarterMaxBranches, 1); err != nil {
		return Plans{}, err
	}
	if plans.Starter.MaxActiveServices, err = requiredInt(get, envStarterMaxServices, 0); err != nil {
		return Plans{}, err
	}
	if plans.Starter.Price, err = requiredInt(get, envStarterPrice, 1); err != nil {
		return Plans{}, err
	}
	if plans.Pro.MaxBranches, err = requiredInt(get, envProMaxBranches, 1); err != nil {
		return Plans{}, err
	}
	if plans.Pro.MaxActiveServices, err = requiredInt(get, envProMaxServices, 0); err != nil {
		return Plans{}, err
	}
	if plans.Pro.Price, err = requiredInt(get, envProPrice, 1); err != nil {
		return Plans{}, err
	}

	// Pro includes everything Starter has, so it can never offer less.
	if plans.Pro.MaxBranches < plans.Starter.MaxBranches {
		return Plans{}, fmt.Errorf("config: %s %d must not be lower than %s %d",
			envProMaxBranches, plans.Pro.MaxBranches, envStarterMaxBranches, plans.Starter.MaxBranches)
	}
	if lowerLimit(plans.Pro.MaxActiveServices, plans.Starter.MaxActiveServices) {
		return Plans{}, fmt.Errorf("config: %s %d must not be lower than %s %d",
			envProMaxServices, plans.Pro.MaxActiveServices, envStarterMaxServices, plans.Starter.MaxActiveServices)
	}

	return plans, nil
}

// lowerLimit reports whether limit a offers less than b, where 0 means unlimited.
func lowerLimit(a, b int) bool {
	if a == 0 {
		return false
	}
	return b == 0 || a < b
}

func requiredInt(get func(string) string, name string, minimum int) (int, error) {
	raw := strings.TrimSpace(get(name))
	if raw == "" {
		return 0, fmt.Errorf("config: %s is required and must not be empty", name)
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("config: %s %q must be a whole number: %w", name, raw, err)
	}
	if value < minimum {
		return 0, fmt.Errorf("config: %s %d must be at least %d", name, value, minimum)
	}
	return value, nil
}
