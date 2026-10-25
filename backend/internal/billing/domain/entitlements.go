// Package domain holds the billing rules the rest of the platform consumes: which
// plans exist and what each one allows. It imports no infrastructure.
package domain

import "errors"

// Errors for plans and their limits.
var (
	ErrPlanInvalid  = errors.New("billing: plan is not valid")
	ErrLimitInvalid = errors.New("billing: limit must not be negative")
)

// Plan is a subscription plan. Any business may take any plan.
type Plan string

// The plans of the platform.
const (
	PlanStarter Plan = "starter"
	PlanPro     Plan = "pro"
)

// ParsePlan converts the stored or transmitted spelling into a Plan.
func ParsePlan(raw string) (Plan, error) {
	switch plan := Plan(raw); plan {
	case PlanStarter, PlanPro:
		return plan, nil
	default:
		return "", ErrPlanInvalid
	}
}

// Entitlements is what a business may have. The rest of the code asks it
// questions such as CanAddBranch and never asks which plan the business is on.
type Entitlements struct {
	maxBranches       int
	maxActiveServices int
}

// NewEntitlements builds the limits of a plan. A limit of 0 means unlimited.
func NewEntitlements(maxBranches, maxActiveServices int) (Entitlements, error) {
	if maxBranches < 0 || maxActiveServices < 0 {
		return Entitlements{}, ErrLimitInvalid
	}
	return Entitlements{maxBranches: maxBranches, maxActiveServices: maxActiveServices}, nil
}

// MaxBranches is the branch limit, 0 for unlimited.
func (e Entitlements) MaxBranches() int { return e.maxBranches }

// MaxActiveServices is the active service limit, 0 for unlimited.
func (e Entitlements) MaxActiveServices() int { return e.maxActiveServices }

// CanAddBranch reports whether a business with current branches may create one more.
func (e Entitlements) CanAddBranch(current int) bool { return fits(current, e.maxBranches) }

// CanActivateService reports whether a business with currentActive active
// services may activate or create one more.
func (e Entitlements) CanActivateService(currentActive int) bool {
	return fits(currentActive, e.maxActiveServices)
}

// BranchesToDeactivate applies the downgrade policy to branches: nothing is
// deleted, the oldest ones stay active up to the limit and the rest are returned
// so the caller marks them inactive. activeOldestFirst must be ordered oldest first.
func (e Entitlements) BranchesToDeactivate(activeOldestFirst []int64) []int64 {
	return overflow(activeOldestFirst, e.maxBranches)
}

// ServicesToDeactivate is the same policy for active services.
func (e Entitlements) ServicesToDeactivate(activeOldestFirst []int64) []int64 {
	return overflow(activeOldestFirst, e.maxActiveServices)
}

func fits(current, limit int) bool { return limit == 0 || current < limit }

func overflow(oldestFirst []int64, limit int) []int64 {
	if limit == 0 || len(oldestFirst) <= limit {
		return nil
	}
	return append([]int64(nil), oldestFirst[limit:]...)
}
