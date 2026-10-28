package postgres_test

import (
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

func TestMonthlyUsageCountsGeneratedAndRedeemedIndependently(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newShop(ctx, t, pool, "studio", "active", 0)
	other := newShop(ctx, t, pool, "other", "active", 0)
	owner, _ := ownerOf(ctx, t, pool, s.business)
	store := postgres.NewCoupons(pool)
	start, end := domain.QuotaMonth(now)

	// Generated this month: A and B. Generated last month and redeemed this month: C.
	for _, c := range []struct{ phone, code string }{{"5491100000001", "AAAAAAAA"}, {"5491100000002", "BBBBBBBB"}} {
		if _, err := store.Issue(ctx, request(s, c.phone, c.code)); err != nil {
			t.Fatalf("issue %s: %v", c.code, err)
		}
	}
	lastMonth := request(s, "5491100000003", "CCCCCCCC")
	lastMonth.Now = now.AddDate(0, -1, 0)
	lastMonth.ExpiresAt = now.Add(48 * time.Hour)
	if _, err := store.Issue(ctx, lastMonth); err != nil {
		t.Fatalf("issue last month: %v", err)
	}
	if _, err := store.Issue(ctx, request(other, "5491100000009", "DDDDDDDD")); err != nil {
		t.Fatalf("issue other: %v", err)
	}
	for _, code := range []string{"AAAAAAAA", "CCCCCCCC"} {
		if _, err := store.Redeem(ctx, redeemRequest(s, owner, code)); err != nil {
			t.Fatalf("redeem %s: %v", code, err)
		}
	}

	got, err := store.MonthlyUsage(ctx, s.business, start, end)
	if err != nil {
		t.Fatalf("MonthlyUsage: %v", err)
	}
	if u := got[s.benefit]; u.Generated != 2 || u.Redeemed != 2 {
		t.Errorf("usage = %+v, want 2 generated (A, B) and 2 redeemed (A, and C from last month)", u)
	}
	if len(got) != 1 {
		t.Errorf("usage = %v, want only the benefits of this business with coupons", got)
	}
}

func TestOverviewCountsTheCouponsOfTodayAndTheMetricsOfTheLastSevenDays(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newShop(ctx, t, pool, "studio", "active", 0)
	other := newShop(ctx, t, pool, "other", "active", 0)
	owner, branch := ownerOf(ctx, t, pool, s.business)
	store := postgres.NewCoupons(pool)

	// now is 15 March 09:00 in Argentina. A is redeemed, B is still pending, C expired a
	// minute ago, and the other business has a pending one that must not show.
	for _, c := range []struct{ phone, code string }{{"5491100000001", "AAAAAAAA"}, {"5491100000002", "BBBBBBBB"}, {"5491100000003", "CCCCCCCC"}} {
		if _, err := store.Issue(ctx, request(s, c.phone, c.code)); err != nil {
			t.Fatalf("issue %s: %v", c.code, err)
		}
	}
	if _, err := store.Issue(ctx, request(other, "5491100000009", "DDDDDDDD")); err != nil {
		t.Fatalf("issue other: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE benefit_codes SET expires_at = $1 WHERE code = 'CCCCCCCC'`, now.Add(-time.Minute)); err != nil {
		t.Fatalf("expire C: %v", err)
	}
	if _, err := store.Redeem(ctx, redeemRequest(s, owner, "AAAAAAAA")); err != nil {
		t.Fatalf("redeem: %v", err)
	}

	// The last 7 days are 9 to 15 March: the 8th is out, so is the other business.
	for _, m := range []struct {
		business, branch any
		date, metric     string
		count            int
	}{
		{s.business, nil, "2028-03-15", "profile_view", 10},
		{s.business, nil, "2028-03-09", "profile_view", 5},
		{s.business, nil, "2028-03-08", "profile_view", 100},
		{s.business, branch, "2028-03-15", "whatsapp_click", 3},
		{s.business, branch, "2028-03-12", "whatsapp_click", 4},
		{s.business, branch, "2028-03-08", "whatsapp_click", 100},
		{other.business, nil, "2028-03-15", "profile_view", 1000},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO metric_daily (business_id, branch_id, date, metric, count) VALUES ($1, $2, $3::date, $4, $5)`,
			m.business, m.branch, m.date, m.metric, m.count); err != nil {
			t.Fatalf("insert metric: %v", err)
		}
	}

	got, err := store.Overview(ctx, s.business, now)
	if err != nil {
		t.Fatalf("Overview: %v", err)
	}
	if got.ActiveCoupons != 1 || got.RedeemedToday != 1 || got.ProfileViews != 15 || got.WhatsAppClicks != 7 {
		t.Errorf("overview = %+v, want 1 active (B), 1 redeemed today, 15 views and 7 clicks", got)
	}

	// Tomorrow at 09:00 the redemption is no longer "today".
	if tomorrow, err := store.Overview(ctx, s.business, now.Add(24*time.Hour)); err != nil || tomorrow.RedeemedToday != 0 {
		t.Errorf("tomorrow = %+v, %v; want nothing redeemed today", tomorrow, err)
	}
	empty := newShop(ctx, t, pool, "empty", "active", 0)
	if zero, err := store.Overview(ctx, empty.business, now); err != nil || zero.ActiveCoupons != 0 || zero.RedeemedToday != 0 || zero.ProfileViews != 0 || zero.WhatsAppClicks != 0 {
		t.Errorf("a business with nothing = %+v, %v; want zeros", zero, err)
	}
}
