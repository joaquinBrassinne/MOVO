package migrations_test

import (
	"fmt"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

// TestVisibleBranchesExposeOnlyActiveBranchesOfActiveBusinesses pins the minimal
// visibility rule: a branch is public if it is active and its business is active.
func TestVisibleBranchesExposeOnlyActiveBranchesOfActiveBusinesses(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()

	visible := func(branchID int64) bool {
		t.Helper()
		var ok bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM visible_branches WHERE branch_id = $1)`, branchID).Scan(&ok); err != nil {
			t.Fatalf("query visible_branches: %v", err)
		}
		return ok
	}

	for i, tc := range []struct {
		status       string
		branchActive bool
		want         bool
	}{
		{"active", true, true},
		{"draft", true, false},
		{"pending_review", true, false},
		{"paused", true, false},
		{"deleted", true, false},
		{"active", false, false},
	} {
		name := fmt.Sprintf("%s/branch-active=%t", tc.status, tc.branchActive)
		owner := insertOwner(ctx, t, pool, fmt.Sprintf("owner%d@example.com", i))
		businessID, err := insertBusiness(ctx, pool, owner, fmt.Sprintf("studio-%d", i), "5491112345678", tc.status)
		if err != nil {
			t.Fatalf("%s: insert business: %v", name, err)
		}
		branchID := insertBranch(ctx, t, pool, businessID)
		if _, err := pool.Exec(ctx, `UPDATE branches SET active = $2 WHERE id = $1`, branchID, tc.branchActive); err != nil {
			t.Fatalf("%s: set branch active: %v", name, err)
		}

		if got := visible(branchID); got != tc.want {
			t.Errorf("%s: visible = %t, want %t", name, got, tc.want)
		}
	}
}

// TestVisibleBranchesFollowTheLiveState pins that visibility is computed on read:
// pausing the business hides its branch without touching the branch.
func TestVisibleBranchesFollowTheLiveState(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	businessID, err := insertBusiness(ctx, pool, insertOwner(ctx, t, pool, "owner@example.com"), "studio", "5491112345678", "active")
	if err != nil {
		t.Fatalf("insert business: %v", err)
	}
	branchID := insertBranch(ctx, t, pool, businessID)

	count := func() (n int) {
		t.Helper()
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM visible_branches WHERE branch_id = $1`, branchID).Scan(&n); err != nil {
			t.Fatalf("count: %v", err)
		}
		return n
	}
	if got := count(); got != 1 {
		t.Fatalf("active business: %d rows, want 1", got)
	}
	if _, err := pool.Exec(ctx, `UPDATE businesses SET status = 'paused' WHERE id = $1`, businessID); err != nil {
		t.Fatalf("pause: %v", err)
	}
	if got := count(); got != 0 {
		t.Errorf("paused business: %d rows, want 0", got)
	}
}
