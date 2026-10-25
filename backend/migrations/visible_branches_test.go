package migrations_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

// TestVisibleBranchesExposeOnlyActiveBranchesOfActiveBusinesses pins the minimal
// visibility rule: a branch is public if it is active and its business is active.
// pilot never decides it (SDD 4.1: the pilot and the payment join in phase 5).
func TestVisibleBranchesExposeOnlyActiveBranchesOfActiveBusinesses(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()

	for i, tc := range []struct {
		name         string
		status       string
		pilot        bool
		branchActive bool
		want         bool
	}{
		{"active business, active branch", "active", false, true, true},
		{"active pilot business, active branch", "active", true, true, true},
		{"draft business", "draft", false, true, false},
		{"draft pilot business", "draft", true, true, false},
		{"pending review business", "pending_review", false, true, false},
		{"paused business", "paused", false, true, false},
		{"paused pilot business", "paused", true, true, false},
		{"deleted business", "deleted", false, true, false},
		{"inactive branch", "active", false, false, false},
		{"inactive branch of a pilot business", "active", true, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			owner := insertOwner(ctx, t, pool, fmt.Sprintf("owner%d@example.com", i))
			businessID, err := insertBusiness(ctx, pool, owner, fmt.Sprintf("studio-%d", i), "5491112345678", tc.status)
			if err != nil {
				t.Fatalf("insert business: %v", err)
			}
			if _, err := pool.Exec(ctx, `UPDATE businesses SET pilot = $2 WHERE id = $1`, businessID, tc.pilot); err != nil {
				t.Fatalf("set pilot: %v", err)
			}
			branchID := insertBranch(ctx, t, pool, businessID)
			if _, err := pool.Exec(ctx, `UPDATE branches SET active = $2 WHERE id = $1`, branchID, tc.branchActive); err != nil {
				t.Fatalf("set branch active: %v", err)
			}

			var got bool
			if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM visible_branches WHERE branch_id = $1)`, branchID).Scan(&got); err != nil {
				t.Fatalf("query visible_branches: %v", err)
			}
			if got != tc.want {
				t.Errorf("visible = %t, want %t", got, tc.want)
			}
		})
	}
}

// TestVisibleBranchesListOnlyTheActiveBranchesOfABusiness pins that the rule is
// applied per branch: a business with several branches shows just its active ones.
func TestVisibleBranchesListOnlyTheActiveBranchesOfABusiness(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	business, err := insertBusiness(ctx, pool, insertOwner(ctx, t, pool, "owner@example.com"), "studio", "5491112345678", "active")
	if err != nil {
		t.Fatalf("insert business: %v", err)
	}
	first := insertBranch(ctx, t, pool, business)
	second := insertBranch(ctx, t, pool, business)
	third := insertBranch(ctx, t, pool, business)
	if _, err := pool.Exec(ctx, `UPDATE branches SET active = false WHERE id = $1`, second); err != nil {
		t.Fatalf("retire: %v", err)
	}

	rows, err := pool.Query(ctx, `SELECT branch_id FROM visible_branches WHERE business_id = $1 ORDER BY branch_id`, business)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	var got []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got = append(got, id)
	}
	if want := []int64{first, third}; !slices.Equal(got, want) {
		t.Errorf("visible branches = %v, want %v", got, want)
	}
}

// TestVisibleBranchesKeepTheirColumnContract pins the columns and their order. Phase 5
// redefines the view with CREATE OR REPLACE VIEW, which can only keep or append
// columns, and the search queries read these by name; this fails first if either breaks.
func TestVisibleBranchesKeepTheirColumnContract(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	rows, err := pool.Query(t.Context(), `
		SELECT column_name || ':' || udt_name
		FROM information_schema.columns
		WHERE table_name = 'visible_branches'
		ORDER BY ordinal_position`)
	if err != nil {
		t.Fatalf("query columns: %v", err)
	}
	defer rows.Close()

	var got []string
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got = append(got, column)
	}
	want := []string{
		"branch_id:int8", "business_id:int8", "category_id:int8", "business_name:text",
		"business_slug:text", "branch_name:text", "address:text", "city:text", "reference:text",
		"location:geography", "rating_avg:numeric", "rating_count:int4", "verified_at:timestamptz",
		"featured:bool",
	}
	if !slices.Equal(got, want) {
		t.Errorf("columns = %v\nwant     %v", got, want)
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
