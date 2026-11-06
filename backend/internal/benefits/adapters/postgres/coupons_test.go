package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

// shop is a business with one active branch and one benefit.
type shop struct{ business, benefit int64 }

// newShop creates a business in the given status with an active branch, so it is visible
// when active, and a percent benefit. The business caps its monthly coupons at limit;
// 0 means no limit.
func newShop(ctx context.Context, t *testing.T, pool *pgxpool.Pool, name, status string, limit int) shop {
	t.Helper()

	var owner, business, benefit int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, role) VALUES ($1, 'hash', 'business_admin') RETURNING id`, name+"@example.com").Scan(&owner); err != nil {
		t.Fatalf("insert owner: %v", err)
	}
	if err := pool.QueryRow(ctx, `
INSERT INTO businesses (owner_user_id, category_id, name, slug, whatsapp_number, status, monthly_coupon_limit, pilot)
VALUES ($1, (SELECT id FROM categories WHERE slug = 'yoga'), $2, $2, '5491112345678', $3, NULLIF($4, 0), true) RETURNING id`, owner, name, status, limit).Scan(&business); err != nil {
		t.Fatalf("insert business: %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO branches (business_id, name, address, city, location)
VALUES ($1, 'Centro', 'Calle 1', 'Cordoba', ST_SetSRID(ST_MakePoint(-64.18, -31.42), 4326)::geography)`, business); err != nil {
		t.Fatalf("insert branch: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO benefits (business_id, type, percent) VALUES ($1, 'percent', 10) RETURNING id`, business).Scan(&benefit); err != nil {
		t.Fatalf("insert benefit: %v", err)
	}
	return shop{business: business, benefit: benefit}
}

var now = time.Date(2028, 3, 15, 12, 0, 0, 0, time.UTC)

// raceCode returns a distinct valid code for each n: the alphabet has no 0, O, 1 or I.
func raceCode(n int) string {
	const letters = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	return "ZZZZZZ" + string(letters[n/len(letters)]) + string(letters[n%len(letters)])
}

// request builds an issue request for a phone, with a code and token derived from it.
func request(s shop, phone, code string) benefits.IssueRequest {
	start, end := domain.QuotaMonth(now)
	hash := domain.HashAccessToken(phone + code)
	return benefits.IssueRequest{
		BenefitID:  s.benefit,
		Contact:    domain.Contact{Name: "Ana", Phone: phone},
		Code:       code,
		TokenHash:  hash,
		Now:        now,
		ExpiresAt:  domain.ExpiryFrom(now),
		MonthStart: start, MonthEnd: end,
	}
}

func TestIssueStoresAPendingCouponForANewCustomer(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newShop(ctx, t, pool, "studio", "active", 5)

	req := request(s, "5491112345678", "ABCDEFGH")
	got, err := postgres.NewCoupons(pool).Issue(ctx, req)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if got.Coupon.ID == 0 || got.Coupon.Status != domain.StatusPending || got.Coupon.BusinessID != s.business || got.Benefit.Percent != 10 {
		t.Errorf("issued = %+v", got)
	}

	var (
		status, name string
		hash         []byte
		expires      time.Time
	)
	if err := pool.QueryRow(ctx, `
SELECT c.status, cu.name, c.access_token_hash, c.expires_at
FROM benefit_codes c JOIN customers cu ON cu.id = c.customer_id WHERE c.code = 'ABCDEFGH'`).Scan(&status, &name, &hash, &expires); err != nil {
		t.Fatalf("read coupon: %v", err)
	}
	if status != "pending" || name != "Ana" || string(hash) != string(req.TokenHash) || !expires.Equal(now.Add(48*time.Hour)) {
		t.Errorf("stored = %q %q expiring %v, want pending, Ana, the hash and +48h", status, name, expires)
	}
}

func TestIssueOnlyServesAnActiveBenefitOfAVisibleBusiness(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	issuer := postgres.NewCoupons(pool)
	draft := newShop(ctx, t, pool, "draft-shop", "draft", 5)
	paused := newShop(ctx, t, pool, "paused-shop", "paused", 5)
	retired := newShop(ctx, t, pool, "retired-shop", "active", 5)
	off := newShop(ctx, t, pool, "off-shop", "active", 5)
	if _, err := pool.Exec(ctx, `UPDATE branches SET active = false WHERE business_id = $1`, retired.business); err != nil {
		t.Fatalf("retire branch: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE benefits SET active = false WHERE id = $1`, off.benefit); err != nil {
		t.Fatalf("deactivate benefit: %v", err)
	}

	for name, s := range map[string]shop{"draft business": draft, "paused business": paused, "no visible branch": retired, "inactive benefit": off, "unknown benefit": {benefit: 999999}} {
		if _, err := issuer.Issue(ctx, request(s, "5491112345678", "ABCDEFGH")); !errors.Is(err, benefits.ErrBenefitNotFound) {
			t.Errorf("%s: error = %v, want ErrBenefitNotFound", name, err)
		}
	}
	var customers int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM customers`).Scan(&customers); err != nil || customers != 0 {
		t.Errorf("customers after refused requests = %d, %v; want none filed", customers, err)
	}
}

func TestIssueGivesOnePendingCouponPerCustomerAndBusiness(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	issuer := postgres.NewCoupons(pool)
	mine := newShop(ctx, t, pool, "mine", "active", 5)
	other := newShop(ctx, t, pool, "other", "active", 5)

	if _, err := issuer.Issue(ctx, request(mine, "5491112345678", "AAAAAAAA")); err != nil {
		t.Fatalf("first coupon: %v", err)
	}
	if _, err := issuer.Issue(ctx, request(mine, "5491112345678", "BBBBBBBB")); !errors.Is(err, benefits.ErrPendingCoupon) {
		t.Errorf("second coupon at the same business: error = %v, want ErrPendingCoupon", err)
	}
	if _, err := issuer.Issue(ctx, request(mine, "5491187654321", "CCCCCCCC")); err != nil {
		t.Errorf("another customer: %v", err)
	}
	if _, err := issuer.Issue(ctx, request(other, "5491112345678", "DDDDDDDD")); err != nil {
		t.Errorf("same customer at another business: %v", err)
	}

	// Redeeming frees the slot.
	if _, err := pool.Exec(ctx, `
UPDATE benefit_codes SET status = 'redeemed', redeemed_at = now(), redeemed_by = (SELECT owner_user_id FROM businesses WHERE id = $1),
       redeemed_branch_id = (SELECT id FROM branches WHERE business_id = $1) WHERE code = 'AAAAAAAA'`, mine.business); err != nil {
		t.Fatalf("redeem: %v", err)
	}
	if _, err := issuer.Issue(ctx, request(mine, "5491112345678", "EEEEEEEE")); err != nil {
		t.Errorf("after redeeming: %v", err)
	}
}

func TestIssueFreesTheSlotOfAnOverdueCouponTheSweepHasNotMarkedYet(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	issuer := postgres.NewCoupons(pool)
	s := newShop(ctx, t, pool, "studio", "active", 5)

	old := request(s, "5491112345678", "AAAAAAAA")
	old.Now, old.ExpiresAt = now.Add(-72*time.Hour), now.Add(-24*time.Hour)
	if _, err := issuer.Issue(ctx, old); err != nil {
		t.Fatalf("old coupon: %v", err)
	}

	if _, err := issuer.Issue(ctx, request(s, "5491112345678", "BBBBBBBB")); err != nil {
		t.Fatalf("a customer with only an overdue coupon must get a new one: %v", err)
	}
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM benefit_codes WHERE code = 'AAAAAAAA'`).Scan(&status); err != nil || status != "expired" {
		t.Errorf("overdue coupon status = %q, %v; want expired", status, err)
	}
}

func TestIssueStopsAtTheMonthlyLimitOfTheBusinessAndCountsOnlyThisMonth(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	issuer := postgres.NewCoupons(pool)
	s := newShop(ctx, t, pool, "studio", "active", 2)
	other := newShop(ctx, t, pool, "other", "active", 1)

	// Coupons from last month do not count.
	if _, err := issuer.Issue(ctx, request(s, "5491100000001", "AAAAAAAA")); err != nil {
		t.Fatalf("old coupon: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE benefit_codes SET created_at = $1, status = 'expired' WHERE code = 'AAAAAAAA'`, now.AddDate(0, -1, 0)); err != nil {
		t.Fatalf("age coupon: %v", err)
	}
	// Both of this month's slots are free, whatever happened to those coupons.
	for i, phone := range []string{"5491100000002", "5491100000003"} {
		if _, err := issuer.Issue(ctx, request(s, phone, fmt.Sprintf("BBBBBBB%d", i+2))); err != nil {
			t.Fatalf("coupon %d of the month: %v", i+1, err)
		}
	}
	if _, err := issuer.Issue(ctx, request(s, "5491100000004", "CCCCCCCC")); !errors.Is(err, benefits.ErrQuotaReached) {
		t.Errorf("third coupon of the month: error = %v, want ErrQuotaReached", err)
	}
	// The limit counts generated coupons, redeemed or not, and belongs to the business.
	if _, err := issuer.Issue(ctx, request(other, "5491100000004", "DDDDDDDD")); err != nil {
		t.Errorf("another business has its own limit: %v", err)
	}
}

func TestIssueNeverOvershootsTheLimitWhenCustomersRace(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	issuer := postgres.NewCoupons(pool)
	const limit, racers = 3, 12
	s := newShop(ctx, t, pool, "studio", "active", limit)

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		issued   int
		refused  int
		failures []error
	)
	for i := range racers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := issuer.Issue(ctx, request(s, fmt.Sprintf("54911000000%02d", i), raceCode(i)))
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				issued++
			case errors.Is(err, benefits.ErrQuotaReached):
				refused++
			default:
				failures = append(failures, err)
			}
		}()
	}
	wg.Wait()

	if issued != limit || refused != racers-limit || len(failures) != 0 {
		t.Errorf("issued %d, refused %d, failures %v; want exactly %d issued and %d refused", issued, refused, failures, limit, racers-limit)
	}
}

func TestIssueGivesOneCouponWhenTheSameCustomerRaces(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	issuer := postgres.NewCoupons(pool)
	s := newShop(ctx, t, pool, "studio", "active", 50)

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		issued  int
		pending int
		other   []error
	)
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := issuer.Issue(ctx, request(s, "5491112345678", raceCode(i)))
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				issued++
			case errors.Is(err, benefits.ErrPendingCoupon):
				pending++
			default:
				other = append(other, err)
			}
		}()
	}
	wg.Wait()

	if issued != 1 || pending != 7 || len(other) != 0 {
		t.Errorf("issued %d, pending %d, other %v; want 1 issued and 7 refused", issued, pending, other)
	}
}

func TestIssueReportsACodeCollisionAndKeepsTheCustomersName(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	issuer := postgres.NewCoupons(pool)
	s := newShop(ctx, t, pool, "studio", "active", 5)

	if _, err := issuer.Issue(ctx, request(s, "5491100000001", "ABCDEFGH")); err != nil {
		t.Fatalf("first coupon: %v", err)
	}
	if _, err := issuer.Issue(ctx, request(s, "5491100000002", "ABCDEFGH")); !errors.Is(err, benefits.ErrCodeTaken) {
		t.Errorf("same code: error = %v, want ErrCodeTaken", err)
	}
	// The failed attempt rolled back: the same customer can retry with a new code.
	if _, err := issuer.Issue(ctx, request(s, "5491100000002", "JKLMNPQR")); err != nil {
		t.Errorf("retry with another code: %v", err)
	}

	// One phone is one customer: a coupon at another business reuses the row and keeps the
	// name declared first. A refused request rolls back, so it leaves the name alone.
	other := newShop(ctx, t, pool, "other", "active", 5)
	renamed := request(other, "5491100000001", "STUVWXYZ")
	renamed.Contact.Name = "Ana Maria"
	if _, err := issuer.Issue(ctx, renamed); err != nil {
		t.Fatalf("same customer at another business: %v", err)
	}
	refused := request(s, "5491100000001", "STUVWXYY")
	refused.Contact.Name = "Someone Else"
	if _, err := issuer.Issue(ctx, refused); !errors.Is(err, benefits.ErrPendingCoupon) {
		t.Fatalf("customer with a pending coupon: error = %v, want ErrPendingCoupon", err)
	}

	var customers int
	var name string
	if err := pool.QueryRow(ctx, `SELECT count(*), max(name) FROM customers WHERE phone = '5491100000001'`).Scan(&customers, &name); err != nil {
		t.Fatalf("read customer: %v", err)
	}
	if customers != 1 || name != "Ana" {
		t.Errorf("customers for the phone = %d named %q; want one named Ana", customers, name)
	}
}
