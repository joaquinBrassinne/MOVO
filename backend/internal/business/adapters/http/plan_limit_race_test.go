package businesshttp_test

import (
	"net/http"
	"sync"
	"testing"
)

// racers is how many requests hit the plan limit at the same moment.
const racers = 12

// race sends the same request racers times at once and counts the outcomes.
func race(do func() reply) (created, refused int) {
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		release = make(chan struct{})
	)
	for range racers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-release
			got := do()

			mu.Lock()
			defer mu.Unlock()
			switch {
			case got.status < 300:
				created++
			case got.status == http.StatusConflict && got.errorCode() == "plan_limit_reached":
				refused++
			}
		}()
	}
	close(release)
	wg.Wait()
	return created, refused
}

// TestConcurrentBranchesNeverPassThePlanLimit pins that counting the active
// branches and storing the new one is a single step: with room for one, a burst
// of requests stores exactly one and refuses the rest.
func TestConcurrentBranchesNeverPassThePlanLimit(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{branches: 1})
	owner := ownerWithBusiness(s, "owner@example.com")

	created, refused := race(func() reply {
		return s.do(http.MethodPost, "/me/business/branches", owner, validBranch())
	})
	if created != 1 || refused != racers-1 {
		t.Fatalf("branches stored = %d, refused = %d, want 1 and %d", created, refused, racers-1)
	}
}

// TestConcurrentServicesNeverPassThePlanLimit is the same guarantee for services.
func TestConcurrentServicesNeverPassThePlanLimit(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{services: 2})
	owner := ownerWithBusiness(s, "owner@example.com")

	created, refused := race(func() reply {
		return s.do(http.MethodPost, "/me/business/services", owner, map[string]any{"name": "Yoga"})
	})
	if created != 2 || refused != racers-2 {
		t.Fatalf("services stored = %d, refused = %d, want 2 and %d", created, refused, racers-2)
	}
}

// TestConcurrentActivationsNeverPassThePlanLimit pins the same step for turning
// inactive services on: only the ones that fit become active.
func TestConcurrentActivationsNeverPassThePlanLimit(t *testing.T) {
	t.Parallel()

	s := newServer(t, planLimits{services: 1})
	owner := ownerWithBusiness(s, "owner@example.com")

	var paths []string
	for range racers {
		made := s.do(http.MethodPost, "/me/business/services", owner, map[string]any{"name": "Yoga"})
		if made.status != http.StatusCreated {
			t.Fatalf("create service = %d %v", made.status, made.body)
		}
		path := "/me/business/services/" + serviceID(made)
		if got := s.do(http.MethodPut, path+"/active", owner, map[string]any{"active": false}); got.status != http.StatusOK {
			t.Fatalf("deactivate = %d", got.status)
		}
		paths = append(paths, path)
	}

	var next int
	var pick sync.Mutex
	created, refused := race(func() reply {
		pick.Lock()
		path := paths[next]
		next++
		pick.Unlock()
		return s.do(http.MethodPut, path+"/active", owner, map[string]any{"active": true})
	})
	if created != 1 || refused != racers-1 {
		t.Fatalf("services activated = %d, refused = %d, want 1 and %d", created, refused, racers-1)
	}
}
