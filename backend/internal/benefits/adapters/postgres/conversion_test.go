package postgres_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

var argentina = time.FixedZone("ART", -3*60*60)

// periodEnding is the window of days days that ends with the Argentine day `last`, built the
// way the query builds it.
func periodEnding(last time.Time, days int) benefits.MetricsPeriod {
	to := time.Date(last.Year(), last.Month(), last.Day(), 0, 0, 0, 0, argentina).AddDate(0, 0, 1)
	from := to.AddDate(0, 0, -days)
	return benefits.MetricsPeriod{
		Days: days, From: from, To: to,
		FromDay: from.Format(time.DateOnly), ToDay: to.AddDate(0, 0, -1).Format(time.DateOnly),
	}
}

type conversionFixture struct {
	pool     *pgxpool.Pool
	business int64
	owner    int64
	branch   int64
	customer int64
	n        int
}

func newConversionFixture(t *testing.T, name string) *conversionFixture {
	t.Helper()
	ctx := t.Context()
	pool := dbtest.New(t)
	s := newShop(ctx, t, pool, name, "active", 0)
	f := &conversionFixture{pool: pool, business: s.business}
	if err := pool.QueryRow(ctx, `SELECT owner_user_id FROM businesses WHERE id = $1`, s.business).Scan(&f.owner); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT id FROM branches WHERE business_id = $1`, s.business).Scan(&f.branch); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO customers (name, phone) VALUES ('Ana', '5491100000001') RETURNING id`).Scan(&f.customer); err != nil {
		t.Fatal(err)
	}
	return f
}

// coupon inserts an expired or redeemed coupon created at the instant given. A redeemed one is
// redeemed two hours later at branch.
func (f *conversionFixture) coupon(t *testing.T, createdAt time.Time, redeemed bool, branch int64) {
	t.Helper()
	f.n++
	code := raceCode(f.n)
	hash := make([]byte, 32)
	copy(hash, code)
	var benefit int64
	if err := f.pool.QueryRow(t.Context(), `SELECT id FROM benefits WHERE business_id = $1`, f.business).Scan(&benefit); err != nil {
		t.Fatal(err)
	}
	var err error
	if redeemed {
		_, err = f.pool.Exec(t.Context(), `
INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, status, expires_at, created_at, redeemed_at, redeemed_by, redeemed_branch_id)
VALUES ($1, $2, $3, $4, $5, 'redeemed', $6, $6, $6::timestamptz + interval '2 hours', $7, $8)`,
			code, benefit, f.business, f.customer, hash, createdAt, f.owner, branch)
	} else {
		_, err = f.pool.Exec(t.Context(), `
INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, status, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5, 'expired', $6, $6)`,
			code, benefit, f.business, f.customer, hash, createdAt)
	}
	if err != nil {
		t.Fatalf("insert coupon: %v", err)
	}
}

func (f *conversionFixture) metric(t *testing.T, metric, day string, branch, service int64, n int) {
	t.Helper()
	var branchArg, serviceArg any
	if branch != 0 {
		branchArg = branch
	}
	if service != 0 {
		serviceArg = service
	}
	if _, err := f.pool.Exec(t.Context(), `
INSERT INTO metric_daily (business_id, branch_id, service_id, date, metric, count) VALUES ($1, $2, $3, $4::date, $5, $6)`,
		f.business, branchArg, serviceArg, day, metric, n); err != nil {
		t.Fatalf("insert metric: %v", err)
	}
}

// TestConversionCountsRespectTheArgentineDayAndMonthBoundaries pins the window edges: a coupon
// at 23:59:59 of the day before the period belongs to the previous one, one at 00:00:00 of the
// first day to this one, and the one at 00:00:00 of the day after the period to neither.
func TestConversionCountsRespectTheArgentineDayAndMonthBoundaries(t *testing.T) {
	t.Parallel()

	f := newConversionFixture(t, "studio")
	current := periodEnding(time.Date(2028, 3, 15, 0, 0, 0, 0, argentina), 7) // March 9 to 15
	previous := periodEnding(time.Date(2028, 3, 8, 0, 0, 0, 0, argentina), 7) // March 2 to 8
	at := func(y int, m time.Month, d, h, mi, s int) time.Time {
		return time.Date(y, m, d, h, mi, s, 0, argentina)
	}

	f.coupon(t, at(2028, 3, 8, 23, 59, 59), true, f.branch)   // previous, redeemed
	f.coupon(t, at(2028, 3, 2, 0, 0, 0), false, f.branch)     // previous, first instant
	f.coupon(t, at(2028, 3, 1, 23, 59, 59), true, f.branch)   // before the previous period
	f.coupon(t, at(2028, 3, 9, 0, 0, 0), true, f.branch)      // current, first instant
	f.coupon(t, at(2028, 3, 15, 23, 59, 59), false, f.branch) // current, last instant
	f.coupon(t, at(2028, 3, 16, 0, 0, 0), true, f.branch)     // after the period

	// Views of the profile on the edge days, across the end of February in a leap year.
	f.metric(t, "profile_view", "2028-03-01", 0, 0, 100) // before the previous period
	f.metric(t, "profile_view", "2028-03-02", 0, 0, 3)   // previous, first day
	f.metric(t, "profile_view", "2028-03-08", 0, 0, 4)   // previous, last day
	f.metric(t, "profile_view", "2028-03-09", 0, 0, 5)   // current, first day
	f.metric(t, "profile_view", "2028-03-15", 0, 0, 6)   // current, last day
	f.metric(t, "profile_view", "2028-03-16", 0, 0, 100) // after the period

	got, err := postgres.NewCoupons(f.pool).Conversion(t.Context(), f.business, current, previous)
	if err != nil {
		t.Fatal(err)
	}
	if want := (benefits.ConversionCounts{ProfileViews: 11, CouponsGenerated: 2, CouponsRedeemed: 1}); got.Current != want {
		t.Errorf("current = %+v, want %+v", got.Current, want)
	}
	if want := (benefits.ConversionCounts{ProfileViews: 7, CouponsGenerated: 2, CouponsRedeemed: 1}); got.Previous != want {
		t.Errorf("previous = %+v, want %+v", got.Previous, want)
	}
}

func TestConversionByMonthEndCountsFebruary29(t *testing.T) {
	t.Parallel()

	f := newConversionFixture(t, "studio")
	period := periodEnding(time.Date(2028, 3, 2, 0, 0, 0, 0, argentina), 7) // February 25 to March 2
	f.metric(t, "profile_view", "2028-02-24", 0, 0, 100)
	f.metric(t, "profile_view", "2028-02-29", 0, 0, 2)
	f.metric(t, "profile_view", "2028-03-01", 0, 0, 3)
	f.metric(t, "profile_view", "2028-03-02", 0, 0, 4)
	f.metric(t, "profile_view", "2028-03-03", 0, 0, 100)

	got, err := postgres.NewCoupons(f.pool).Conversion(t.Context(), f.business, period, periodEnding(time.Date(2028, 2, 24, 0, 0, 0, 0, argentina), 7))
	if err != nil {
		t.Fatal(err)
	}
	if got.Current.ProfileViews != 9 {
		t.Errorf("views = %d, want 9 (February 29, March 1 and March 2)", got.Current.ProfileViews)
	}
}

func TestConversionBreaksDownByBranchAndActivityForThisBusinessOnly(t *testing.T) {
	t.Parallel()

	f := newConversionFixture(t, "studio")
	ctx := t.Context()
	var second, yoga, pilates, gone int64
	if err := f.pool.QueryRow(ctx, `
INSERT INTO branches (business_id, name, address, city, location)
VALUES ($1, 'Norte', 'Calle 2', 'Cordoba', ST_SetSRID(ST_MakePoint(-64.19, -31.41), 4326)::geography) RETURNING id`, f.business).Scan(&second); err != nil {
		t.Fatal(err)
	}
	insertService := func(dst *int64, name string, deleted bool) {
		if err := f.pool.QueryRow(ctx, `INSERT INTO services (business_id, name, deleted_at) VALUES ($1, $2, CASE WHEN $3 THEN now() END) RETURNING id`,
			f.business, name, deleted).Scan(dst); err != nil {
			t.Fatal(err)
		}
	}
	insertService(&yoga, "Yoga", false)
	insertService(&pilates, "Pilates", false)
	insertService(&gone, "Borrado", true)

	current := periodEnding(time.Date(2028, 3, 15, 0, 0, 0, 0, argentina), 7)
	previous := periodEnding(time.Date(2028, 3, 8, 0, 0, 0, 0, argentina), 7)
	inside := time.Date(2028, 3, 12, 10, 0, 0, 0, argentina)

	f.coupon(t, inside, true, f.branch)
	f.coupon(t, inside, true, f.branch)
	f.coupon(t, inside, true, second)
	f.coupon(t, inside, false, second)
	f.metric(t, "whatsapp_click", "2028-03-12", f.branch, 0, 10)
	f.metric(t, "whatsapp_click", "2028-03-12", f.branch, yoga, 5)
	f.metric(t, "whatsapp_click", "2028-03-13", second, pilates, 4)
	f.metric(t, "service_view", "2028-03-12", f.branch, yoga, 20)
	f.metric(t, "service_view", "2028-03-12", second, pilates, 8)
	f.metric(t, "service_view", "2028-03-12", f.branch, gone, 999)
	f.metric(t, "whatsapp_click", "2028-03-01", f.branch, yoga, 999) // outside the period

	// Another business with its own numbers must not leak in.
	other := newShop(ctx, t, f.pool, "other", "active", 0)
	var otherBranch int64
	if err := f.pool.QueryRow(ctx, `SELECT id FROM branches WHERE business_id = $1`, other.business).Scan(&otherBranch); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `INSERT INTO metric_daily (business_id, branch_id, date, metric, count) VALUES ($1, $2, '2028-03-12', 'whatsapp_click', 777)`, other.business, otherBranch); err != nil {
		t.Fatal(err)
	}

	got, err := postgres.NewCoupons(f.pool).Conversion(ctx, f.business, current, previous)
	if err != nil {
		t.Fatal(err)
	}

	wantBranches := map[string]benefits.BranchConversion{
		"Centro": {BranchID: f.branch, Name: "Centro", WhatsAppClicks: 15, Redemptions: 2},
		"Norte":  {BranchID: second, Name: "Norte", WhatsAppClicks: 4, Redemptions: 1},
	}
	if len(got.ByBranch) != len(wantBranches) {
		t.Fatalf("branches = %+v, want only this business's two", got.ByBranch)
	}
	for _, b := range got.ByBranch {
		if b != wantBranches[b.Name] {
			t.Errorf("branch %s = %+v, want %+v", b.Name, b, wantBranches[b.Name])
		}
	}

	wantActivities := []benefits.ActivityConversion{
		{ServiceID: pilates, Name: "Pilates", Views: 8, WhatsAppClicks: 4},
		{ServiceID: yoga, Name: "Yoga", Views: 20, WhatsAppClicks: 5},
	}
	if fmt.Sprint(got.ByActivity) != fmt.Sprint(wantActivities) {
		t.Errorf("activities = %+v, want %+v (no deleted service, no click outside the period)", got.ByActivity, wantActivities)
	}
}
