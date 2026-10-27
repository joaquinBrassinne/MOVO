package postgres_test

import (
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

func TestPreviewReadsTheCouponOfTheBusinessAndChangesNothing(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	mine := newShop(ctx, t, pool, "mine", "active", 5)
	other := newShop(ctx, t, pool, "other", "active", 5)
	owner, _ := ownerOf(ctx, t, pool, mine.business)
	store := postgres.NewCoupons(pool)
	if _, err := store.Issue(ctx, request(mine, "5491112345678", "ABCDEFGH")); err != nil {
		t.Fatalf("issue: %v", err)
	}
	if _, err := store.Issue(ctx, request(other, "5491187654321", "JKLMNPQR")); err != nil {
		t.Fatalf("issue other: %v", err)
	}

	got, err := store.Preview(ctx, mine.business, "ABCDEFGH")
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if got.Coupon.Code != "ABCDEFGH" || got.Coupon.Status != domain.StatusPending || got.CustomerName != "Ana" ||
		got.Benefit.Type != domain.TypePercent || got.Benefit.Percent != 10 || got.RedeemedAt != nil || got.Coupon.BusinessID != mine.business {
		t.Errorf("preview = %+v", got)
	}

	// Looking never redeems: the coupon is still pending and can be redeemed afterwards.
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM benefit_codes WHERE code = 'ABCDEFGH'`).Scan(&status); err != nil || status != "pending" {
		t.Fatalf("status after preview = %q, %v; want pending", status, err)
	}
	redeemed, err := store.Redeem(ctx, redeemRequest(mine, owner, "ABCDEFGH"))
	if err != nil {
		t.Fatalf("redeem after preview: %v", err)
	}
	after, err := store.Preview(ctx, mine.business, "ABCDEFGH")
	if err != nil || after.Coupon.Status != domain.StatusRedeemed || after.RedeemedAt == nil || !after.RedeemedAt.Equal(redeemed.RedeemedAt) {
		t.Errorf("preview after redeeming = %+v, %v; want redeemed with its time", after, err)
	}

	// The other business's code is not visible from here, and an unknown one answers the same.
	for _, code := range []string{"JKLMNPQR", "STUVWXYZ"} {
		if _, err := store.Preview(ctx, mine.business, code); !errors.Is(err, benefits.ErrCouponNotFound) {
			t.Errorf("Preview(%s) error = %v, want ErrCouponNotFound", code, err)
		}
	}
}
