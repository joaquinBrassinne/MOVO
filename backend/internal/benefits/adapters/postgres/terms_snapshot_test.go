package postgres_test

import (
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

// TestACouponKeepsTheTermsItWasIssuedUnder pins that editing a benefit changes the
// coupons issued from then on and never the ones already in a customer's hands: what the
// customer sees, what the owner previews, what is redeemed and the activity all read the
// terms stored with the coupon.
func TestACouponKeepsTheTermsItWasIssuedUnder(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newShop(ctx, t, pool, "studio", "active", 0)
	owner, _ := ownerOf(ctx, t, pool, s.business)
	store := postgres.NewCoupons(pool)

	first := request(s, "5491112345678", "ABCDEFGH")
	if _, err := store.Issue(ctx, first); err != nil {
		t.Fatalf("issue: %v", err)
	}

	// The owner edits the benefit: a different kind, with conditions.
	if _, err := pool.Exec(ctx,
		`UPDATE benefits SET type = 'custom', percent = NULL, title = 'Clase gratis', conditions = 'Solo martes' WHERE id = $1`, s.benefit); err != nil {
		t.Fatalf("edit benefit: %v", err)
	}

	held, err := store.ByToken(ctx, first.TokenHash)
	if err != nil {
		t.Fatalf("ByToken: %v", err)
	}
	preview, err := store.Preview(ctx, s.business, "ABCDEFGH")
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	for name, b := range map[string]domain.Benefit{"customer view": held.Benefit, "owner preview": preview.Benefit} {
		if b.Type != domain.TypePercent || b.Percent != 10 || b.Title != "" || b.Conditions != "" {
			t.Errorf("%s = %+v, want the 10%% the coupon was issued with", name, b)
		}
	}
	if held.Benefit.ID != s.benefit {
		t.Errorf("benefit id = %d, want %d: the coupon still points at its benefit", held.Benefit.ID, s.benefit)
	}

	redeemed, err := store.Redeem(ctx, redeemRequest(s, owner, "ABCDEFGH"))
	if err != nil {
		t.Fatalf("Redeem: %v", err)
	}
	if redeemed.Benefit.Type != domain.TypePercent || redeemed.Benefit.Percent != 10 {
		t.Errorf("redeemed benefit = %+v, want the 10%% the coupon was issued with", redeemed.Benefit)
	}

	activity, err := store.RecentActivity(ctx, s.business, nil, 10)
	if err != nil {
		t.Fatalf("RecentActivity: %v", err)
	}
	if len(activity.Events) != 2 {
		t.Fatalf("events = %d, want the coupon's generation and redemption", len(activity.Events))
	}
	for _, e := range activity.Events {
		if e.Benefit.Type != domain.TypePercent || e.Benefit.Percent != 10 {
			t.Errorf("%s event shows %+v, want the 10%% the coupon was issued with", e.Kind, e.Benefit)
		}
	}

	// A coupon issued after the edit carries the new terms.
	second := request(s, "5491187654321", "JKLMNPQR")
	if _, err := store.Issue(ctx, second); err != nil {
		t.Fatalf("issue after edit: %v", err)
	}
	got, err := store.ByToken(ctx, second.TokenHash)
	if err != nil {
		t.Fatalf("ByToken after edit: %v", err)
	}
	if got.Benefit.Type != domain.TypeCustom || got.Benefit.Title != "Clase gratis" || got.Benefit.Conditions != "Solo martes" {
		t.Errorf("new coupon benefit = %+v, want the edited terms", got.Benefit)
	}
}
