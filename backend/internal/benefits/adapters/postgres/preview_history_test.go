package postgres_test

import (
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

// redeemAt issues nothing: it redeems a coupon already issued, at the given moment.
func redeemAt(s shop, owner int64, code string, at time.Time) benefits.RedeemRequest {
	return benefits.RedeemRequest{BusinessID: s.business, UserID: owner, Code: code, Now: at}
}

// TestPreviewListsWhatTheCustomerRedeemedBefore pins that the owner sees which coupons the
// person already redeemed, with the terms each one had, where and when, and not only how
// many. Another customer, another business and the coupon being looked at stay out.
func TestPreviewListsWhatTheCustomerRedeemedBefore(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	mine := newShop(ctx, t, pool, "mine", "active", 0)
	other := newShop(ctx, t, pool, "other", "active", 0)
	owner, _ := ownerOf(ctx, t, pool, mine.business)
	otherOwner, _ := ownerOf(ctx, t, pool, other.business)
	store := postgres.NewCoupons(pool)

	const ana, luis = "5491112345678", "5491187654321"
	earlier := now.Add(-2 * time.Hour)

	// Ana redeems a 10% coupon; Luis and Ana at the other business do the same.
	for _, c := range []struct {
		s     shop
		by    int64
		phone string
		code  string
	}{{mine, owner, ana, raceCode(1)}, {mine, owner, luis, raceCode(2)}, {other, otherOwner, ana, raceCode(3)}} {
		if _, err := store.Issue(ctx, request(c.s, c.phone, c.code)); err != nil {
			t.Fatalf("issue %s: %v", c.code, err)
		}
		if _, err := store.Redeem(ctx, redeemAt(c.s, c.by, c.code, earlier)); err != nil {
			t.Fatalf("redeem %s: %v", c.code, err)
		}
	}

	// The benefit changes, and Ana comes back for a new coupon.
	if _, err := pool.Exec(ctx,
		`UPDATE benefits SET type = 'custom', percent = NULL, title = 'Clase gratis' WHERE id = $1`, mine.benefit); err != nil {
		t.Fatalf("edit benefit: %v", err)
	}
	if _, err := store.Issue(ctx, request(mine, ana, raceCode(4))); err != nil {
		t.Fatalf("issue again: %v", err)
	}

	got, err := store.Preview(ctx, mine.business, raceCode(4))
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if got.CustomerRedeemed != 1 || len(got.PreviousRedemptions) != 1 {
		t.Fatalf("redeemed before = %d, listed = %d, want exactly Ana's one at this business", got.CustomerRedeemed, len(got.PreviousRedemptions))
	}
	past := got.PreviousRedemptions[0]
	if past.Benefit.Type != domain.TypePercent || past.Benefit.Percent != 10 {
		t.Errorf("past benefit = %+v, want the 10%% it had when it was redeemed, not today's terms", past.Benefit)
	}
	if past.BranchName != "Centro" || !past.RedeemedAt.Equal(earlier) {
		t.Errorf("past = %q at %v, want Centro at %v", past.BranchName, past.RedeemedAt, earlier)
	}

	// A coupon that is already redeemed does not list itself, but still counts.
	self, err := store.Preview(ctx, mine.business, raceCode(1))
	if err != nil {
		t.Fatalf("Preview redeemed: %v", err)
	}
	if len(self.PreviousRedemptions) != 0 || self.CustomerRedeemed != 1 {
		t.Errorf("redeemed coupon: listed = %d, count = %d, want none listed and the count unchanged", len(self.PreviousRedemptions), self.CustomerRedeemed)
	}
}

// TestPreviewListsOnlyTheLatestRedemptionsNewestFirst pins the bound of the list: the
// count says how many there were, the list shows the newest ones.
func TestPreviewListsOnlyTheLatestRedemptionsNewestFirst(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newShop(ctx, t, pool, "mine", "active", 0)
	owner, _ := ownerOf(ctx, t, pool, s.business)
	store := postgres.NewCoupons(pool)

	const phone, total = "5491112345678", 12
	for i := range total {
		code := raceCode(i)
		if _, err := store.Issue(ctx, request(s, phone, code)); err != nil {
			t.Fatalf("issue %d: %v", i, err)
		}
		// Hours before now, the oldest first: coupon 0 is the oldest redemption.
		at := now.Add(-time.Duration(total-i) * time.Hour)
		if _, err := store.Redeem(ctx, redeemAt(s, owner, code, at)); err != nil {
			t.Fatalf("redeem %d: %v", i, err)
		}
	}
	if _, err := store.Issue(ctx, request(s, phone, raceCode(total))); err != nil {
		t.Fatalf("issue the last: %v", err)
	}

	got, err := store.Preview(ctx, s.business, raceCode(total))
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if got.CustomerRedeemed != total || len(got.PreviousRedemptions) != benefits.PreviousRedemptionsLimit {
		t.Fatalf("count = %d, listed = %d, want %d and %d", got.CustomerRedeemed, len(got.PreviousRedemptions), total, benefits.PreviousRedemptionsLimit)
	}
	newest, oldest := got.PreviousRedemptions[0], got.PreviousRedemptions[len(got.PreviousRedemptions)-1]
	if want := now.Add(-1 * time.Hour); !newest.RedeemedAt.Equal(want) {
		t.Errorf("first listed = %v, want the newest, %v", newest.RedeemedAt, want)
	}
	if want := now.Add(-time.Duration(total-2) * time.Hour); !oldest.RedeemedAt.Equal(want) {
		t.Errorf("last listed = %v, want %v: the two oldest are left out", oldest.RedeemedAt, want)
	}
}
