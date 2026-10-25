package migrations_test

import (
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

// TestMetricDailyShapeFollowsTheMetric pins the rule kept in the database: a profile
// view has no branch and no service, a click needs its branch, a service view needs
// both.
func TestMetricDailyShapeFollowsTheMetric(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	business, err := insertBusiness(ctx, pool, insertOwner(ctx, t, pool, "owner@example.com"), "studio", "5491112345678", "active")
	if err != nil {
		t.Fatalf("insert business: %v", err)
	}
	branch := insertBranch(ctx, t, pool, business)
	var service int64
	if err := pool.QueryRow(ctx, `INSERT INTO services (business_id, name) VALUES ($1, 'Yoga') RETURNING id`, business).Scan(&service); err != nil {
		t.Fatalf("insert service: %v", err)
	}

	insert := func(metric string, branchID, serviceID any, day int) error {
		_, err := pool.Exec(ctx,
			`INSERT INTO metric_daily (business_id, branch_id, service_id, date, metric, count)
			 VALUES ($1, $2, $3, make_date(2026, 3, $4), $5, 1)`, business, branchID, serviceID, day, metric)
		return err
	}

	for i, tc := range []struct {
		name            string
		metric          string
		branch, service any
		wantCheck       bool
	}{
		{"profile view alone", "profile_view", nil, nil, false},
		{"profile view with a branch", "profile_view", branch, nil, true},
		{"profile view with a service", "profile_view", nil, service, true},
		{"click with its branch", "whatsapp_click", branch, nil, false},
		{"click with branch and service", "whatsapp_click", branch, service, false},
		{"click without a branch", "whatsapp_click", nil, nil, true},
		{"service view with both", "service_view", branch, service, false},
		{"service view without a service", "service_view", branch, nil, true},
		{"service view without a branch", "service_view", nil, service, true},
		{"unknown metric", "page_view", branch, nil, true},
	} {
		// A new day per case keeps the unique key out of the way of this test.
		err := insert(tc.metric, tc.branch, tc.service, i+1)
		if tc.wantCheck {
			requireSQLState(t, err, checkViolation)
		} else if err != nil {
			t.Errorf("%s: must be allowed: %v", tc.name, err)
		}
	}
}

// TestMetricDailyKeyTreatsNullsAsEqual pins that the same counter cannot be inserted
// twice even when its branch or service is NULL, which a plain UNIQUE would allow.
func TestMetricDailyKeyTreatsNullsAsEqual(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	business, err := insertBusiness(ctx, pool, insertOwner(ctx, t, pool, "owner@example.com"), "studio", "5491112345678", "active")
	if err != nil {
		t.Fatalf("insert business: %v", err)
	}
	branch := insertBranch(ctx, t, pool, business)

	for _, tc := range []struct {
		name, metric string
		branch       any
	}{
		{"profile view, NULL branch and service", "profile_view", nil},
		{"click, NULL service", "whatsapp_click", branch},
	} {
		insert := func() error {
			_, err := pool.Exec(ctx,
				`INSERT INTO metric_daily (business_id, branch_id, date, metric, count)
				 VALUES ($1, $2, '2026-03-01', $3, 1)`, business, tc.branch, tc.metric)
			return err
		}
		if err := insert(); err != nil {
			t.Fatalf("%s: first insert: %v", tc.name, err)
		}
		requireSQLState(t, insert(), uniqueViolation)
	}
}

// TestMetricDailyKeepsBranchAndServiceInsideTheirBusiness pins that a counter cannot
// point at a branch or a service of another business, and that deleting the business
// deletes its counters.
func TestMetricDailyKeepsBranchAndServiceInsideTheirBusiness(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	mine, err := insertBusiness(ctx, pool, insertOwner(ctx, t, pool, "mine@example.com"), "mine", "5491112345678", "active")
	if err != nil {
		t.Fatalf("insert business: %v", err)
	}
	other, err := insertBusiness(ctx, pool, insertOwner(ctx, t, pool, "other@example.com"), "other", "5491112345679", "active")
	if err != nil {
		t.Fatalf("insert other business: %v", err)
	}
	myBranch := insertBranch(ctx, t, pool, mine)
	otherBranch := insertBranch(ctx, t, pool, other)
	var otherService int64
	if err := pool.QueryRow(ctx, `INSERT INTO services (business_id, name) VALUES ($1, 'Pilates') RETURNING id`, other).Scan(&otherService); err != nil {
		t.Fatalf("insert service: %v", err)
	}

	_, err = pool.Exec(ctx, `INSERT INTO metric_daily (business_id, branch_id, date, metric, count) VALUES ($1, $2, '2026-03-01', 'whatsapp_click', 1)`, mine, otherBranch)
	requireSQLState(t, err, foreignKeyViolation)
	_, err = pool.Exec(ctx, `INSERT INTO metric_daily (business_id, branch_id, service_id, date, metric, count) VALUES ($1, $2, $3, '2026-03-01', 'service_view', 1)`, mine, myBranch, otherService)
	requireSQLState(t, err, foreignKeyViolation)

	if _, err := pool.Exec(ctx, `INSERT INTO metric_daily (business_id, branch_id, date, metric, count) VALUES ($1, $2, '2026-03-01', 'whatsapp_click', 1)`, mine, myBranch); err != nil {
		t.Fatalf("own branch must be allowed: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM businesses WHERE id = $1`, mine); err != nil {
		t.Fatalf("delete business: %v", err)
	}
	var left int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM metric_daily WHERE business_id = $1`, mine).Scan(&left); err != nil || left != 0 {
		t.Errorf("counters left after deleting the business = %d, %v; want 0", left, err)
	}
}
