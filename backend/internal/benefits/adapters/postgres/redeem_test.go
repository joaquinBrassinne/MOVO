package postgres_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

// ownerOf returns the owner user of a business and its only branch.
func ownerOf(ctx context.Context, t *testing.T, pool *pgxpool.Pool, business int64) (owner, branch int64) {
	t.Helper()
	if err := pool.QueryRow(ctx, `SELECT owner_user_id FROM businesses WHERE id = $1`, business).Scan(&owner); err != nil {
		t.Fatalf("find owner: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT min(id) FROM branches WHERE business_id = $1`, business).Scan(&branch); err != nil {
		t.Fatalf("find branch: %v", err)
	}
	return owner, branch
}

func addBranch(ctx context.Context, t *testing.T, pool *pgxpool.Pool, business int64, active bool) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(ctx, `
INSERT INTO branches (business_id, name, address, city, location, active)
VALUES ($1, 'Norte', 'Calle 2', 'Cordoba', ST_SetSRID(ST_MakePoint(-64.19, -31.40), 4326)::geography, $2) RETURNING id`, business, active).Scan(&id); err != nil {
		t.Fatalf("insert branch: %v", err)
	}
	return id
}

func redeemRequest(s shop, user int64, code string) benefits.RedeemRequest {
	return benefits.RedeemRequest{BusinessID: s.business, UserID: user, Code: code, Now: now}
}

func TestRedeemMarksTheCouponAndReturnsWhatTheBusinessNeeds(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newShop(ctx, t, pool, "studio", "active", 5)
	owner, branch := ownerOf(ctx, t, pool, s.business)
	store := postgres.NewCoupons(pool)
	if _, err := store.Issue(ctx, request(s, "5491112345678", "ABCDEFGH")); err != nil {
		t.Fatalf("issue: %v", err)
	}

	got, err := store.Redeem(ctx, redeemRequest(s, owner, "ABCDEFGH"))
	if err != nil {
		t.Fatalf("Redeem: %v", err)
	}
	if got.Coupon.Code != "ABCDEFGH" || got.Coupon.Status != domain.StatusRedeemed || got.CustomerName != "Ana" ||
		got.Benefit.Type != domain.TypePercent || got.Benefit.Percent != 10 || got.BranchID != branch || !got.RedeemedAt.Equal(now) {
		t.Errorf("redeemed = %+v", got)
	}

	var (
		status         string
		by, atBranch   int64
		redeemedAtTime time.Time
	)
	if err := pool.QueryRow(ctx,
		`SELECT status, redeemed_by, redeemed_branch_id, redeemed_at FROM benefit_codes WHERE code = 'ABCDEFGH'`).Scan(&status, &by, &atBranch, &redeemedAtTime); err != nil {
		t.Fatalf("read coupon: %v", err)
	}
	if status != "redeemed" || by != owner || atBranch != branch || !redeemedAtTime.Equal(now) {
		t.Errorf("stored = %q by %d at branch %d on %v; want redeemed by the owner at the branch", status, by, atBranch, redeemedAtTime)
	}
}

func TestRedeemSaysWhyACouponCannotBeRedeemed(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	mine := newShop(ctx, t, pool, "mine", "active", 5)
	other := newShop(ctx, t, pool, "other", "active", 5)
	owner, _ := ownerOf(ctx, t, pool, mine.business)
	store := postgres.NewCoupons(pool)

	for i, code := range []string{"AAAAAAAA", "BBBBBBBB", "CCCCCCCC", "DDDDDDDD"} {
		if _, err := store.Issue(ctx, request(mine, "549110000000"+string(rune('1'+i)), code)); err != nil {
			t.Fatalf("issue %s: %v", code, err)
		}
	}
	if _, err := store.Issue(ctx, request(other, "5491199999999", "ZZZZZZZZ")); err != nil {
		t.Fatalf("issue other: %v", err)
	}
	if _, err := store.Redeem(ctx, redeemRequest(mine, owner, "AAAAAAAA")); err != nil {
		t.Fatalf("redeem: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE benefit_codes SET status = 'expired' WHERE code = 'BBBBBBBB'`); err != nil {
		t.Fatalf("expire: %v", err)
	}

	late := redeemRequest(mine, owner, "CCCCCCCC")
	late.Now = now.Add(48 * time.Hour) // pending in the table, but past its expiry

	for name, tc := range map[string]struct {
		req  benefits.RedeemRequest
		want error
	}{
		"already redeemed":         {redeemRequest(mine, owner, "AAAAAAAA"), benefits.ErrCouponRedeemed},
		"marked expired":           {redeemRequest(mine, owner, "BBBBBBBB"), benefits.ErrCouponExpired},
		"overdue, not swept yet":   {late, benefits.ErrCouponExpired},
		"code of another business": {redeemRequest(mine, owner, "ZZZZZZZZ"), benefits.ErrCouponNotFound},
		"code that does not exist": {redeemRequest(mine, owner, "JKLMNPQR"), benefits.ErrCouponNotFound},
	} {
		if _, err := store.Redeem(ctx, tc.req); !errors.Is(err, tc.want) {
			t.Errorf("%s: error = %v, want %v", name, err, tc.want)
		}
	}

	// Nothing was changed by the refusals: the other business's coupon is still pending,
	// and the overdue one was not marked by a read.
	for code, want := range map[string]string{"ZZZZZZZZ": "pending", "CCCCCCCC": "pending", "DDDDDDDD": "pending"} {
		var status string
		if err := pool.QueryRow(ctx, `SELECT status FROM benefit_codes WHERE code = $1`, code).Scan(&status); err != nil || status != want {
			t.Errorf("%s status = %q, %v; want %q", code, status, err, want)
		}
	}
}

func TestRedeemResolvesTheBranch(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newShop(ctx, t, pool, "studio", "active", 9)
	stranger := newShop(ctx, t, pool, "stranger", "active", 9)
	owner, first := ownerOf(ctx, t, pool, s.business)
	_, foreign := ownerOf(ctx, t, pool, stranger.business)
	store := postgres.NewCoupons(pool)
	issue := func(phone, code string) {
		t.Helper()
		if _, err := store.Issue(ctx, request(s, phone, code)); err != nil {
			t.Fatalf("issue %s: %v", code, err)
		}
	}
	redeem := func(code string, branch int64) (benefits.Redeemed, error) {
		req := redeemRequest(s, owner, code)
		req.BranchID = branch
		return store.Redeem(ctx, req)
	}

	// One active branch: it is the default. A retired one does not count.
	retired := addBranch(ctx, t, pool, s.business, false)
	issue("5491100000001", "AAAAAAAA")
	if got, err := redeem("AAAAAAAA", 0); err != nil || got.BranchID != first {
		t.Fatalf("default branch = %+v, %v; want the only active branch %d", got, err, first)
	}

	// Two active branches: the owner has to say which.
	second := addBranch(ctx, t, pool, s.business, true)
	issue("5491100000002", "BBBBBBBB")
	if _, err := redeem("BBBBBBBB", 0); !errors.Is(err, benefits.ErrBranchRequired) {
		t.Errorf("no branch with two active = %v, want ErrBranchRequired", err)
	}
	for name, id := range map[string]int64{"a retired branch": retired, "a branch of another business": foreign, "a branch that does not exist": 999999} {
		if _, err := redeem("BBBBBBBB", id); !errors.Is(err, benefits.ErrBranchNotFound) {
			t.Errorf("%s = %v, want ErrBranchNotFound", name, err)
		}
	}
	if got, err := redeem("BBBBBBBB", second); err != nil || got.BranchID != second {
		t.Errorf("explicit branch = %+v, %v; want %d", got, err, second)
	}

	// The refusals above left the coupon redeemable until the right call.
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM benefit_codes WHERE code = 'BBBBBBBB'`).Scan(&status); err != nil || status != "redeemed" {
		t.Errorf("status = %q, %v", status, err)
	}
}

// TestRedeemRedeemsExactlyOnceWhenManyTryAtTheSameTime is the mandatory concurrency test
// of the coupon rule (CLAUDE.md): any number of simultaneous redemptions of one code
// produce exactly one success.
func TestRedeemRedeemsExactlyOnceWhenManyTryAtTheSameTime(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newShop(ctx, t, pool, "studio", "active", 5)
	owner, _ := ownerOf(ctx, t, pool, s.business)
	store := postgres.NewCoupons(pool)
	if _, err := store.Issue(ctx, request(s, "5491112345678", "ABCDEFGH")); err != nil {
		t.Fatalf("issue: %v", err)
	}

	const racers = 40
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		succeeded int
		redeemed  int
		others    []error
		start     = make(chan struct{})
	)
	for range racers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := store.Redeem(ctx, redeemRequest(s, owner, "ABCDEFGH"))
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				succeeded++
			case errors.Is(err, benefits.ErrCouponRedeemed):
				redeemed++
			default:
				others = append(others, err)
			}
		}()
	}
	close(start)
	wg.Wait()

	if succeeded != 1 || redeemed != racers-1 || len(others) != 0 {
		t.Errorf("succeeded %d, already redeemed %d, other %v; want exactly 1 success and %d refusals", succeeded, redeemed, others, racers-1)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM benefit_codes WHERE code = 'ABCDEFGH' AND status = 'redeemed'`).Scan(&count); err != nil || count != 1 {
		t.Errorf("redeemed rows = %d, %v; want 1", count, err)
	}
}
